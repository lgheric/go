ps -ef | grep order| awk '{print $2}' | xargs -I X kill -9 X
echo "building ..."
go build
./order -s start >> log/log.log &
./order -s end >> log/log.log &
ps -ef | grep order
