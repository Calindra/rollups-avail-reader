package paioavail

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/calindra/rollups-base-reader/pkg/contracts"
	"github.com/calindra/rollups-base-reader/pkg/eip712"
	"github.com/calindra/rollups-base-reader/pkg/inputreader"
	"github.com/calindra/rollups-base-reader/pkg/model"
	"github.com/calindra/rollups-base-reader/pkg/paiodecoder"
	"github.com/calindra/rollups-base-reader/pkg/services"
	"github.com/calindra/rollups-base-reader/pkg/supervisor"
	cModel "github.com/cartesi/rollups-graphql/pkg/convenience/model"
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	TIMESTAMP_SECTION_INDEX = 3
	DELAY                   = 500
	ONE_SECOND_IN_MS        = 1000
	FIVE_MINUTES            = 300
)

type AvailListener struct {
	PaioDecoder       paiodecoder.DecoderPaio
	InputService      *services.InputService
	InputReaderWorker *inputreader.InputReaderWorker
	FromBlock         uint64
	AvailFromBlock    uint64
	L1CurrentBlock    uint64
	L1ReadDelay       int
}

type PaioDecoder interface {
	DecodePaioBatch(ctx context.Context, bytes []byte) (string, error)
}

func NewAvailListener(availFromBlock uint64, inputService *services.InputService,
	w *inputreader.InputReaderWorker, fromBlock uint64, binaryDecoderPathLocation string,
	applicationAddress string,
) supervisor.Worker {
	var paioDecoder PaioDecoder = paiodecoder.ZzzzHuiDecoder{}
	if binaryDecoderPathLocation != "" {
		paioDecoder = paiodecoder.NewPaioDecoder(binaryDecoderPathLocation)
	}
	l1ReadDelay := FIVE_MINUTES
	l1ReadDelayStr, ok := os.LookupEnv("L1_READ_DELAY_IN_SECONDS")
	if ok {
		aux, err := strconv.Atoi(l1ReadDelayStr)
		if err != nil {
			slog.Error("Configuration error: The L1_READ_DELAY_IN_SECONDS environment variable should be a numeric value.")
			panic(err)
		}
		l1ReadDelay = aux
	}
	return AvailListener{
		AvailFromBlock:    availFromBlock,
		InputService:      inputService,
		InputReaderWorker: w,
		L1CurrentBlock:    fromBlock,
		L1ReadDelay:       l1ReadDelay,
		PaioDecoder:       paioDecoder,
	}
}

func (a AvailListener) String() string {
	return "avail_listener"
}

func (a AvailListener) Start(ctx context.Context, ready chan<- struct{}) error {
	ready <- struct{}{}
	client, err := a.connect(ctx)
	if err != nil {
		slog.Error("Avail", "Error connecting to Avail", err)
		return err
	}
	return a.watchNewTransactions(ctx, client)
}

func (a AvailListener) connect(ctx context.Context) (*gsrpc.SubstrateAPI, error) {
	rpcURL, haveURL := os.LookupEnv("AVAIL_RPC_URL")
	if !haveURL {
		rpcURL = DEFAULT_AVAIL_RPC_URL
	}

	errCh := make(chan error)
	clientCh := make(chan *gsrpc.SubstrateAPI)

	go func() {
		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
			default:
				client, err := NewSubstrateAPICtx(ctx, rpcURL)
				if err != nil {
					slog.Error("Avail", "Error connecting to Avail client", err)
					slog.Info("Avail reconnecting client", "retryInterval", retryInterval)
					time.Sleep(retryInterval)
				} else {
					clientCh <- client
					return
				}

			}
		}
	}()

	select {
	case err := <-errCh:
		return nil, err
	case client := <-clientCh:
		return client, nil
	}
}

const retryInterval = 5 * time.Second

func (a AvailListener) watchNewTransactions(ctx context.Context, client *gsrpc.SubstrateAPI) error {
	latestAvailBlock := a.AvailFromBlock
	var index uint = 0
	defer client.Client.Close()

	ethClient, err := a.InputReaderWorker.GetEthClient()
	if err != nil {
		return fmt.Errorf("avail input reader: dial: %w", err)
	}
	inputBox, err := contracts.NewInputBox(a.InputReaderWorker.InputBoxAddress, ethClient)
	if err != nil {
		return fmt.Errorf("avail input reader: bind input box: %w", err)
	}

	for {
		if latestAvailBlock == 0 {
			block, err := client.RPC.Chain.GetHeaderLatest()
			if err != nil {
				slog.Error("Avail", "Error getting latest block hash", err)
				slog.Info("Avail reconnecting", "retryInterval", retryInterval)
				time.Sleep(retryInterval)
				continue
			}

			slog.Info("Avail", "Set last block", block.Number)
			latestAvailBlock = uint64(block.Number)
		}

		subscription, err := client.RPC.Chain.SubscribeNewHeads()
		if err != nil {
			slog.Error("Avail", "Error subscribing to new heads", err)
			slog.Info("Avail reconnecting", "retryInterval", retryInterval)
			time.Sleep(retryInterval)
			continue
		}
		defer subscription.Unsubscribe()

		errCh := make(chan error)

		go func() {
			for {
				select {
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				case err := <-subscription.Err():
					errCh <- err
					return
				case <-time.After(DELAY * time.Millisecond):

				case i := <-subscription.Chan():
					for latestAvailBlock <= uint64(i.Number) {
						index++

						if latestAvailBlock < uint64(i.Number) {
							slog.Debug("Avail Catching up", "Chain is at block", i.Number, "fetching block", latestAvailBlock)
						} else {
							slog.Debug("Avail", "index", index, "Chain is at block", i.Number, "fetching block", latestAvailBlock)
						}

						blockHash, err := client.RPC.Chain.GetBlockHash(latestAvailBlock)
						if err != nil {
							errCh <- err
							return
						}
						block, err := client.RPC.Chain.GetBlock(blockHash)
						if err != nil {
							errCh <- err
							return
						}
						currentL1Block, err := a.TableTennis(ctx, block,
							ethClient, inputBox,
							a.L1CurrentBlock,
						)
						if err != nil {
							errCh <- err
							return
						}
						if currentL1Block != nil && *currentL1Block > 0 {
							a.L1CurrentBlock = *currentL1Block
						}
						latestAvailBlock += 1
						time.Sleep(500 * time.Millisecond) // nolint
					}
				}
			}
		}()

		err = <-errCh
		subscription.Unsubscribe()

		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err != nil {
			slog.Error("Avail", "Error", err)
			slog.Info("Avail reconnecting", "retryInterval", retryInterval)
			time.Sleep(retryInterval)
		} else {
			return nil
		}
	}
}

func (a AvailListener) TableTennis(ctx context.Context,
	block *types.SignedBlock, ethClient *ethclient.Client,
	inputBox *contracts.InputBox, startBlockNumber uint64) (*uint64, error) {
	apps, err := a.InputService.AppRepository.FindAllByDA(ctx, model.DataAvailability_Avail)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve applications by avail data availability: %w", err)
	}
	if len(apps) == 0 {
		slog.Debug("No applications found for the specified data availability")
	}
	appAddresses := make([]common.Address, len(apps))
	for i, app := range apps {
		appAddresses[i] = app.IApplicationAddress
	}

	var currentL1Block uint64
	availInputs, err := a.ReadInputsFromPaioBlock(ctx, block)
	if err != nil {
		return nil, err
	}
	var availBlockTimestamp uint64
	if len(availInputs) == 0 {
		availBlockTimestamp, err = ReadTimestampFromBlock(block)
		if err != nil {
			return nil, err
		}
	} else {
		availBlockTimestamp = uint64(availInputs[0].BlockTimestamp.Unix())
	}
	inputsL1, err := a.InputReaderWorker.FindAllInputsByBlockAndTimestampLT(ctx,
		ethClient, inputBox, startBlockNumber,
		(availBlockTimestamp/ONE_SECOND_IN_MS)-uint64(a.L1ReadDelay),
		appAddresses,
	)
	if err != nil {
		return nil, err
	}
	if len(inputsL1) > 0 {
		currentL1Block = inputsL1[len(inputsL1)-1].BlockNumber + 1
	}
	inputs := append(inputsL1, availInputs...)
	if len(inputs) > 0 {
		inputCount, err := a.InputService.InputRepository.Count(ctx, nil)
		if err != nil {
			return nil, err
		}
		for i := range inputs {
			inputExtra := inputs[i]
			inputExtra.Input.Index = uint64(i) + inputCount
			var app *model.Application = nil
			for _, appr := range apps {
				if inputExtra.AppContract.Hex() == appr.IApplicationAddress.Hex() {
					app = &appr
					break
				}
			}
			if app == nil {
				slog.Warn("Skipping input",
					"appContract", inputExtra.AppContract.Hex(),
					"expected", apps,
				)
				continue
			}
			// The chainId information does not come in Paio's batch.
			input := inputExtra.Input
			input.EpochApplicationID = app.ID

			err = a.InputService.CreateInputID(ctx, app.ID, input)
			if err != nil {
				return nil, fmt.Errorf("avail input reader: create input: %w", err)
			}
			slog.Info("Input saved",
				"index", input.Index,
				"appID", input.EpochApplicationID,
				"payload", input.RawData,
			)
		}
	}
	return &currentL1Block, nil
}

func (av *AvailListener) ReadInputsFromPaioBlock(ctx context.Context, block *types.SignedBlock) ([]model.InputExtra, error) {
	inputs := []model.InputExtra{}
	timestamp, err := ReadTimestampFromBlock(block)
	if err != nil {
		return inputs, err
	}
	chainId, err := av.InputReaderWorker.ChainID()
	if err != nil {
		return inputs, err
	}
	for _, ext := range block.Block.Extrinsics {
		appID := ext.Signature.AppID.Int64()
		slog.Debug("debug", "appID", appID, "timestamp", timestamp)
		if appID != DEFAULT_APP_ID {
			// slog.Debug("Skipping", "appID", appID)
			continue
		}
		args := ext.Method.Args
		jsonStr, err := av.PaioDecoder.DecodePaioBatch(ctx, args)
		if err != nil {
			return inputs, err
		}
		parsedInputs, err := paiodecoder.ParsePaioBatchToInputs(jsonStr, chainId)
		if err != nil {
			return inputs, err
		}
		inputs = append(inputs, parsedInputs...)
	}
	for i := range inputs {
		inputs[i].BlockTimestamp = time.Unix(int64(timestamp), 0)
	}
	return inputs, nil
}

func ReadInputsFromAvailBlockZzzHui(block *types.SignedBlock) ([]cModel.AdvanceInput, error) {
	inputs := []cModel.AdvanceInput{}
	timestamp, err := ReadTimestampFromBlock(block)
	if err != nil {
		return inputs, err
	}
	for _, ext := range block.Block.Extrinsics {
		appID := ext.Signature.AppID.Int64()
		slog.Debug("debug", "appID", appID, "timestamp", timestamp)
		if appID != DEFAULT_APP_ID {
			slog.Debug("Skipping", "appID", appID)
			continue
		}
		args := string(ext.Method.Args)

		msgSender, typedData, signature, err := eip712.ExtractSigAndData(args)
		if err != nil {
			return inputs, err
		}
		paioMessage, err := paiodecoder.ParsePaioFrom712Message(typedData)
		if err != nil {
			return inputs, err
		}
		slog.Debug("MsgSender", "value", msgSender)
		inputs = append(inputs, cModel.AdvanceInput{
			Index:                int(0),
			CartesiTransactionId: common.Bytes2Hex(crypto.Keccak256(signature)),
			MsgSender:            msgSender,
			Payload:              common.Bytes2Hex(paioMessage.Payload),
			AppContract:          common.HexToAddress(paioMessage.App),
			AvailBlockNumber:     int(block.Block.Header.Number),
			AvailBlockTimestamp:  time.Unix(int64(timestamp)/ONE_SECOND_IN_MS, 0),
			InputBoxIndex:        -2,
			Type:                 "Avail",
		})
	}
	return inputs, nil
}
