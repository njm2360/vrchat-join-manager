# サービス本体
go build -o vjm-agent.exe ./cmd/client

# デバッグ用単発ランナー
go build -o debug_runner.exe ./cmd/debug_runner
