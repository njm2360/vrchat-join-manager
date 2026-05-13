# VRChat Join Manager

VRChatのjoin/leaveログを収集するWindows エージェントと、データを集約・配信するサーバの2構成。
API定義は [./api/openapi.yaml](./api/openapi.yaml)に集約し、両方ともそこからコード生成する。

## Server

サーバはLinux等で常駐動作する。

```sh
cd server
go mod tidy
make generate   # ../api/openapi.yaml から internal/gen/api.gen.go を生成
make build      # bin/vjm-server を生成
./bin/vjm-server
```

開発時は `make run` でそのまま起動できる。

## Agent

エージェントはWindowsサービスとして動作する。
前提としてインスタンスの誕生から墓場まで、ずっとインスタンスマスターとして滞在する前提。

```sh
cd agent
go mod tidy
make generate   # ../api/openapi.yaml からクライアントコードを生成
make build      # bin/vjm-agent.exe を生成
```

インストーラの作成:

```sh
make installer
```

Inno Setup の `ISCC.exe` が PATH に必要。見つからない場合は上書き指定する。

```sh
make installer ISCC='/mnt/c/Program Files (x86)/Inno Setup 6/ISCC.exe'
```
