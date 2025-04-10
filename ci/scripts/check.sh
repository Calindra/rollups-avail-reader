#!/bin/bash
set -e

MAX_RETRIES=30
for i in $(seq 1 $MAX_RETRIES); do
  RESULT=$(docker exec -i postgres psql -U postgres -d rollupsdb -t -c "SELECT * FROM public.output;")
  if [[ "$RESULT" =~ "deadbeef" ]]; then
    echo "Result found: $RESULT"
    exit 0
  fi
  echo "Result: $RESULT"
  echo "Waiting for result... attempt $i"
  sleep 5
done
echo "Timeout reached: result not found"
exit 1