# Mini Container Runtime

Go言語で実装された学習用の簡易コンテナランタイムです。
Linux Namespace (Namespaces) を使用して、プロセス、ファイルシステム、ネットワークの隔離を実現しています。
コンテナ技術の低レイヤーな仕組み（`CLONE_NEWPID`, `chroot` など）を理解することを目的としています。

## 機能

- **Namespace隔離**:
  - `CLONE_NEWUTS`: ホスト名の隔離
  - `CLONE_NEWPID`: プロセスIDの隔離
  - `CLONE_NEWNS`: マウントポイントの隔離
  - `CLONE_NEWNET`: ネットワークデバイスの隔離
- **ファイルシステム**: `chroot` を使用したルートディレクトリの変更
- **プロセス管理**: `/proc` ファイルシステムのマウント

## 必要要件

- Linux環境 (Mac/Windowsの場合はDockerを使用してください)
- Go 1.24以上
- Docker & Docker Compose (開発環境用)

## クイックスタート

このプロジェクトはLinuxカーネルの機能に依存しているため、MacやWindows上で開発する場合は付属のDocker環境を使用することを推奨します。

### 1. 開発環境の起動

Docker Composeを使用して開発用コンテナを起動し、シェルに入ります。

```bash
docker compose up -d
docker compose exec dev bash
```

### 2. 依存パッケージのインストール

ネットワーク隔離の確認に必要な `ip` コマンドなどをインストールします（初回のみ）。

```bash
apt-get update && apt-get install -y iproute2
```

### 3. ビルド

ソースコードのあるディレクトリに移動し、ビルドを実行します。

```bash
cd mini-container
go build -o mini-container .
```

### 4. コンテナの起動

用意されている `rootfs` (Alpine Linuxベース) を使用して、ミニコンテナを起動します。

```bash
# 構文: ./mini-container run <rootfsパス> <実行コマンド> [引数...]
./mini-container run ../rootfs /bin/sh
```

### 5. 動作確認

コンテナ内部（`[child]` プロセス）で以下のコマンドを実行し、隔離されていることを確認します。

```bash
# プロセスIDが 1 になっていることを確認
/ # echo $$
1

# ホスト名が "mini-container" になっていることを確認
/ # hostname
mini-container

# ネットワークが隔離され、lo (Loopback) のみが見えることを確認
/ # ip link
1: lo: <LOOPBACK,UP,LOWER_UP> ...
```

コンテナを終了するには `exit` を入力します。

## ディレクトリ構成

- `mini-container/`: ランタイムのGoソースコード
- `rootfs/`: コンテナ実行用のルートファイルシステム（Alpine Linux）
- `docs/`: 設計・実装ドキュメント