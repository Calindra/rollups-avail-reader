package paioavail

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calindra/rollups-base-reader/pkg/commons"
	"github.com/calindra/rollups-base-reader/pkg/contracts"
	"github.com/calindra/rollups-base-reader/pkg/devnet"
	"github.com/calindra/rollups-base-reader/pkg/inputreader"
	"github.com/calindra/rollups-base-reader/pkg/model"
	"github.com/calindra/rollups-base-reader/pkg/paiodecoder"
	"github.com/calindra/rollups-base-reader/pkg/repository"
	"github.com/calindra/rollups-base-reader/pkg/services"
	"github.com/calindra/rollups-base-reader/pkg/supervisor"
	"github.com/jmoiron/sqlx"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/suite"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type AvailListenerSuite struct {
	suite.Suite
	fd              paiodecoder.DecoderPaio
	ctx             context.Context
	workerCtx       context.Context
	timeoutCancel   context.CancelFunc
	workerCancel    context.CancelFunc
	workerResult    chan error
	rpcUrl          string
	portCounter     int
	DbFactory       *commons.DbFactory
	appRepository   *repository.AppRepository
	epochRepository *repository.EpochRepository
	inputRepository *repository.InputRepository
	image           *postgres.PostgresContainer
	schemaPath      string
}

func TestAvailListenerSuite(t *testing.T) {
	suite.Run(t, &AvailListenerSuite{})
}

func (s *AvailListenerSuite) SetupSuite() {
	// Fetch schema
	tmpDir, err := os.MkdirTemp("", "schema")
	s.NoError(err)
	s.schemaPath = filepath.Join(tmpDir, "schema.sql")
	schemaFile, err := os.Create(s.schemaPath)
	s.NoError(err)
	defer schemaFile.Close()

	resp, err := http.Get(commons.Schema)
	s.NoError(err)
	defer resp.Body.Close()

	_, err = io.Copy(schemaFile, resp.Body)
	s.NoError(err)
}

func (s *AvailListenerSuite) TestReadTimestampFromBlockError() {
	block := types.SignedBlock{}
	block.Block = types.Block{}
	timestampExtrinsic := types.Extrinsic{}
	block.Block.Extrinsics = append([]types.Extrinsic{}, timestampExtrinsic)
	_, err := ReadTimestampFromBlock(&block)
	s.ErrorContains(err, "block 0 without timestamp")
}

func (s *AvailListenerSuite) TestReadInputsFromBlockZzzHui() {
	block := types.SignedBlock{}
	block.Block = types.Block{}
	timestampExtrinsic := CreateTimestampExtrinsic()
	block.Block.Extrinsics = append([]types.Extrinsic{}, timestampExtrinsic)
	// nolint
	jsonStr := `{"signature":"0x0a1bcb9c208b3e797e1561970322dc6ba7039b2303c5317d5cb0e970a684c6eb0c4a881c993ab2bc00cdbe95c22492dd4299567e0166f9062a731fba77d375531b","typedData":"eyJ0eXBlcyI6eyJDYXJ0ZXNpTWVzc2FnZSI6W3sibmFtZSI6ImFwcCIsInR5cGUiOiJhZGRyZXNzIn0seyJuYW1lIjoibm9uY2UiLCJ0eXBlIjoidWludDY0In0seyJuYW1lIjoibWF4X2dhc19wcmljZSIsInR5cGUiOiJ1aW50MTI4In0seyJuYW1lIjoiZGF0YSIsInR5cGUiOiJzdHJpbmcifV0sIkVJUDcxMkRvbWFpbiI6W3sibmFtZSI6Im5hbWUiLCJ0eXBlIjoic3RyaW5nIn0seyJuYW1lIjoidmVyc2lvbiIsInR5cGUiOiJzdHJpbmcifSx7Im5hbWUiOiJjaGFpbklkIiwidHlwZSI6InVpbnQyNTYifSx7Im5hbWUiOiJ2ZXJpZnlpbmdDb250cmFjdCIsInR5cGUiOiJhZGRyZXNzIn1dfSwicHJpbWFyeVR5cGUiOiJDYXJ0ZXNpTWVzc2FnZSIsImRvbWFpbiI6eyJuYW1lIjoiQXZhaWxNIiwidmVyc2lvbiI6IjEiLCJjaGFpbklkIjoiMHgzZTkiLCJ2ZXJpZnlpbmdDb250cmFjdCI6IjB4Q2NDQ2NjY2NDQ0NDY0NDQ0NDQ2NDY0NjY0NjQ0NDY0NjY2NjY2NjQyIsInNhbHQiOiIifSwibWVzc2FnZSI6eyJhcHAiOiIweGFiNzUyOGJiODYyZmI1N2U4YTJiY2Q1NjdhMmU5MjlhMGJlNTZhNWUiLCJkYXRhIjoiR00iLCJtYXhfZ2FzX3ByaWNlIjoiMTAiLCJub25jZSI6IjEifX0="}`
	extrinsicInput := CreatePaioExtrinsic([]byte(jsonStr))
	block.Block.Extrinsics = append(block.Block.Extrinsics, extrinsicInput)
	/*
		inputFromL1 := ReadInputsFromL1(&block)
		inputs, err := ReadInputsFromAvailBlock(&block)
		for ...
			dbIndex = repo
			repo.Create()
	*/
	inputs, err := ReadInputsFromAvailBlockZzzHui(&block)
	s.NoError(err)
	s.Equal(1, len(inputs))
	s.Equal(common.HexToAddress("0xab7528bb862fb57e8a2bcd567a2e929a0be56a5e"), inputs[0].AppContract)
	s.Equal("GM", string(common.Hex2Bytes(inputs[0].Payload)))
	s.Equal(common.HexToAddress("0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266"), inputs[0].MsgSender)
}

func (s *AvailListenerSuite) TestReadInputsFromBlockPaio() {
	block := types.SignedBlock{}
	block.Block = types.Block{}
	timestampExtrinsic := CreateTimestampExtrinsic()
	block.Block.Extrinsics = append([]types.Extrinsic{}, timestampExtrinsic)
	// nolint
	fromPaio := "0x1463f9725f107358c9115bc9d86c72dd5823e9b1e60114ab7528bb862fb57e8a2bcd567a2e929a0be56a5e000a0d48656c6c6f2c20576f726c643f2076a270f52ade97cd95ef7be45e08ea956bfdaf14b7fc4f8816207fa9eb3a5c17207ccdd94ac1bd86a749b66526fff6579e2b6bf1698e831955332ad9d5ed44da7208000000000000001c"
	extrinsicPaioBlock := CreatePaioExtrinsic(common.Hex2Bytes(fromPaio))
	block.Block.Extrinsics = append(block.Block.Extrinsics, extrinsicPaioBlock)
	availListener := AvailListener{
		PaioDecoder: s.fd,
		InputReaderWorker: &inputreader.InputReaderWorker{
			Provider: s.rpcUrl,
		},
	}
	inputs, err := availListener.ReadInputsFromPaioBlock(s.ctx, &block)
	s.NoError(err)
	s.Equal(1, len(inputs))
}

func (s *AvailListenerSuite) TestTableTennis() {
	dataavailability := model.DataAvailability_Avail

	// List all applications
	apps, err := s.appRepository.List(s.ctx)
	s.NoError(err)
	s.Require().NotEmpty(apps)

	// Change DA from application to Avail
	firstDapp := apps[0]
	err = s.appRepository.UpdateDA(s.ctx, firstDapp.ID, dataavailability)
	s.NoError(err)

	// check if the DA was updated
	apps, err = s.appRepository.FindAllByDA(s.ctx, dataavailability)
	s.NoError(err)
	s.Require().NotEmpty(apps)
	s.Equal(firstDapp.ID, apps[0].ID)
	s.Equal(dataavailability, apps[0].DataAvailability)

	// Convert app addresses to []common.Address
	appAddresses := make([]common.Address, len(apps))
	for i, app := range apps {
		s.NotEqual(0, int(app.EpochLength), "epoch length should not be 0")
		appAddresses[i] = app.IApplicationAddress
	}

	// main address
	appAddress := "0x8e3c7bF65833ccb1755dAB530Ef0405644FE6ae3"
	s.Contains(appAddresses, common.HexToAddress(appAddress))

	ethClient, err := ethclient.DialContext(s.ctx, s.rpcUrl)
	s.NoError(err)
	inputBoxAddress := common.HexToAddress(devnet.InputBoxAddress)
	inputBox, err := contracts.NewInputBox(inputBoxAddress, ethClient)
	s.NoError(err)
	ctx := context.Background()
	err = devnet.AddInput(ctx, s.rpcUrl, common.Hex2Bytes("deadbeef11"), appAddress)
	s.NoError(err)

	l1FinalizedPrevHeight := uint64(1)
	timestamp := uint64(time.Now().UnixMilli())
	inputterWorker := inputreader.InputReaderWorker{
		Model:           nil,
		Provider:        s.rpcUrl,
		InputBoxAddress: inputBoxAddress,
		InputBoxBlock:   1,
	}
	// Avail's block
	block := types.SignedBlock{}
	block.Block = types.Block{}
	timestampExtrinsic := CreateTimestampExtrinsic()
	delta := 350 * 1000
	timestampExtrinsic.Method.Args = encodeCompactU64(timestamp + uint64(delta))
	block.Block.Extrinsics = append([]types.Extrinsic{}, timestampExtrinsic)
	// nolint
	fromPaio := "0x1463f9725f107358c9115bc9d86c72dd5823e9b1e60114"
	fromPaio = fromPaio + strings.ToLower(appAddress)
	fromPaio = fromPaio + "000a0d48656c6c6f2c20576f726c643f2076a270f52ade97cd95ef7be45e08ea956bfdaf14b7fc4f8816207fa9eb3a5c17207ccdd94ac1bd86a749b66526fff6579e2b6bf1698e831955332ad9d5ed44da7208000000000000001c"
	extrinsicPaioBlock := CreatePaioExtrinsic(common.Hex2Bytes(fromPaio))
	block.Block.Extrinsics = append(block.Block.Extrinsics, extrinsicPaioBlock)

	var fd paiodecoder.DecoderPaio = &FakeDecoder{}
	inputService := services.NewInputService(s.inputRepository, s.epochRepository, s.appRepository)

	availListener := AvailListener{
		PaioDecoder:       fd,
		InputReaderWorker: &inputterWorker,
		InputService:      inputService,
	}
	inputs, err := availListener.ReadInputsFromPaioBlock(ctx, &block)
	s.NoError(err)
	s.Equal(1, len(inputs))
	s.Equal(int64(timestamp)+int64(delta), inputs[0].BlockTimestamp.Unix())
	availBlockTimestamp := uint64(inputs[0].BlockTimestamp.Unix())
	inputs, err = inputterWorker.FindAllInputsByBlockAndTimestampLT(ctx, ethClient, inputBox, l1FinalizedPrevHeight, (availBlockTimestamp/1000)-300, appAddresses)
	s.NoError(err)
	s.NotNil(inputs)
	s.Equal(1, len(inputs))

	savedInputsBeforeTableTennis, err := s.inputRepository.FindAll(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Equal(101, int(savedInputsBeforeTableTennis.Total))

	startBlock := 0
	currentL1Block, err := availListener.TableTennis(s.ctx, &block, ethClient, inputBox, uint64(startBlock))
	s.Require().NoError(err)
	s.NotNil(currentL1Block)

	// check if TableTennis has saved the data.
	lastTwoInputs := 2
	savedInputs, err := s.inputRepository.FindAll(ctx, nil, &lastTwoInputs, nil, nil, nil)
	s.NoError(err)
	s.Equal(103, int(savedInputs.Total))

	// check the input from InputBox
	s.Equal(101, int(savedInputs.Rows[0].Index))
	expectPayload := "deadbeef11"
	s.Equal(expectPayload, common.Bytes2Hex(savedInputs.Rows[0].RawData))

	// check the input from Avail
	s.Equal("Hello, World?", string(savedInputs.Rows[1].RawData))
}

type FakeDecoder struct {
}

func (fd *FakeDecoder) DecodePaioBatch(ctx context.Context, bytes []byte) (string, error) {
	appAddress := "0x8e3c7bF65833ccb1755dAB530Ef0405644FE6ae3"
	// nolint
	jsonStr := fmt.Sprintf(`{"sequencer_payment_address":"0x63F9725f107358c9115BC9d86c72dD5823E9B1E6","txs":[{"app":"%s","nonce":0,"max_gas_price":10,"data":[72,101,108,108,111,44,32,87,111,114,108,100,63],"signature":{"r":"0x76a270f52ade97cd95ef7be45e08ea956bfdaf14b7fc4f8816207fa9eb3a5c17","s":"0x7ccdd94ac1bd86a749b66526fff6579e2b6bf1698e831955332ad9d5ed44da72","v":"0x1c"}}]}`, appAddress)
	return jsonStr, nil
}

func CreatePaioExtrinsic(args []byte) types.Extrinsic {
	return types.Extrinsic{
		Method: types.Call{
			Args: args,
			CallIndex: types.CallIndex{
				SectionIndex: 0,
				MethodIndex:  0,
			},
		},
		Signature: types.ExtrinsicSignatureV4{
			AppID: types.UCompact(*big.NewInt(DEFAULT_APP_ID)),
		},
	}
}

func CreateTimestampExtrinsic() types.Extrinsic {
	return types.Extrinsic{
		Method: types.Call{
			Args: common.Hex2Bytes("0b20008c2e9201"),
			CallIndex: types.CallIndex{
				SectionIndex: 3,
				MethodIndex:  0,
			},
		},
	}
}

func (s *AvailListenerSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	var w supervisor.SupervisorWorker
	w.Name = "SupervisorWorker"
	const testTimeout = 1 * time.Minute
	s.portCounter += 1
	s.ctx, s.timeoutCancel = context.WithTimeout(context.Background(), testTimeout)
	s.workerResult = make(chan error)
	s.fd = &FakeDecoder{}
	s.workerCtx, s.workerCancel = context.WithCancel(s.ctx)
	w.Workers = append(w.Workers, devnet.AnvilWorker{
		Address:  devnet.AnvilDefaultAddress,
		Port:     devnet.AnvilDefaultPort + s.portCounter,
		Verbose:  true,
		AnvilCmd: "anvil",
	})

	// Database
	container, err := postgres.Run(s.ctx, commons.DbImage,
		postgres.BasicWaitStrategies(),
		postgres.WithInitScripts(s.schemaPath),
		postgres.WithDatabase(commons.DbName),
		postgres.WithUsername(commons.DbUser),
		postgres.WithPassword(commons.DbPassword),
		testcontainers.WithLogConsumers(&commons.StdoutLogConsumer{}),
	)
	s.NoError(err)
	extraArg := "sslmode=disable"
	connectionStr, err := container.ConnectionString(s.ctx, extraArg)
	s.NoError(err)
	s.image = container
	err = container.Start(s.ctx)
	s.NoError(err)

	db, err := sqlx.ConnectContext(s.ctx, "postgres", connectionStr)
	s.NoError(err)

	s.appRepository = repository.NewAppRepository(db)
	s.inputRepository = repository.NewInputRepository(db)
	s.epochRepository = repository.NewEpochRepository(db)

	s.rpcUrl = fmt.Sprintf("ws://%s:%v", devnet.AnvilDefaultAddress, devnet.AnvilDefaultPort+s.portCounter)
	ready := make(chan struct{})
	go func() {
		s.workerResult <- w.Start(s.workerCtx, ready)
	}()
	select {
	case <-s.ctx.Done():
		s.Fail("context error", s.ctx.Err())
	case err := <-s.workerResult:
		s.Fail("worker exited before being ready", err)
	case <-ready:
		s.T().Log("nonodo ready")
	}
}

func (s *AvailListenerSuite) TearDownTest() {
	if s.DbFactory != nil {
		defer s.DbFactory.Cleanup()
	}
	err := exec.Command("pkill", "anvil").Run()
	s.NoError(err)
	s.workerCancel()
	select {
	case <-s.ctx.Done():
		s.Fail("context error", s.ctx.Err())
	case err := <-s.workerResult:
		s.NoError(err)
	}
	s.timeoutCancel()
}

// nolint
func encodeCompactU64(value uint64) []byte {
	var result []byte

	if value < (1 << 6) { // Single byte (6-bit value)
		result = []byte{byte(value<<2) | 0b00}
	} else if value < (1 << 14) { // Two bytes (14-bit value)
		result = []byte{
			byte((value&0x3F)<<2) | 0b01,
			byte(value >> 6),
		}
	} else if value < (1 << 30) { // Four bytes (30-bit value)
		result = []byte{
			byte((value&0x3F)<<2) | 0b10,
			byte(value >> 6),
			byte(value >> 14),
			byte(value >> 22),
		}
	} else { // Eight bytes (64-bit value)
		result = []byte{
			0b11, // First byte indicates 8-byte encoding
			byte(value),
			byte(value >> 8),
			byte(value >> 16),
			byte(value >> 24),
			byte(value >> 32),
			byte(value >> 40),
			byte(value >> 48),
			byte(value >> 56),
		}
	}

	return result
}
