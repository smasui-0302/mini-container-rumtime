package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// main プログラムのエントリーポイント
// コマンドライン引数に応じてコンテナ作成処理(run)または
// コンテナ内部処理(child)に分岐
func main() {
	if len(os.Args) < 2 {
		panic("expected run or child")
	}

	switch os.Args[1] {
	case "run":
		run()
	case "child":
		child()
	default:
		panic("invalid command")
	}
}

// run 親プロセスとして動作し新しいNamespaceを作成して子プロセスを起動
// /proc/self/exeをchild引数付きで再実行し
// 自身のコードをコンテナ内部の初期プロセスとして利用
func run() {
	if len(os.Args) < 4 {
		usageAndExit("usage: mini-container run <rootfs> <cmd> [args...]")
	}
	// child側にそのまま"<rootfs> <cmd> [args...]"を渡す
	childArgs := append([]string{"child"}, os.Args[2:]...)
	cmd := exec.Command("/proc/self/exe", childArgs...)

	// SysProcAttrを使用して新しいNamespaceを作成
	// CLONE_NEWUTS: ホスト名とドメイン名を分離
	// CLONE_NEWPID: プロセスID空間を分離
	// CLONE_NEWNS:  マウント名前空間を分離
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET,
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("[run] starting child with new namespace...")
	if err := cmd.Run(); err != nil {
		fmt.Println("run err", err)
		os.Exit(1)
	}
}

// child 新しいNamespace内で実行されるプロセス
// コンテナの初期化(ホスト名設定/chroot/procマウント等)を行い
// 最後にユーザー指定のコマンドを実行
func child() {
	if len(os.Args) < 4 {
		usageAndExit("usage: mini-container child <rootfs> <cmd> [args...]")
	}
	fmt.Printf("[child] PID=%d, setting up container...\n", syscall.Getpid())

	rootfs := os.Args[2]
	cmdPath := os.Args[3]
	cmdArgs := os.Args[3:] // cmdPathとArgsを含む

	// 1. ホスト名の設定
	// コンテナ内でのみ有効なホスト名を設定
	if err := syscall.Sethostname([]byte("mini-container")); err != nil {
		fmt.Println("[child] sethostname error: ", err)
		os.Exit(1)
	}

	// 2. loインターフェースの有効化
	// 新しいNetwork NamespaceではloがDOWN状態のためUPにする
	// chroot前なのでホスト(このプログラムを実行している環境)のipコマンドを利用できる
	if err := exec.Command("ip", "link", "set", "lo", "up").Run(); err != nil {
		fmt.Println("[child] ip link set lo up error: ", err)
		os.Exit(1)
	}

	// 3. ルートファイルシステムの変更(chroot)
	// プロセスのルートディレクトリを指定されたrootfsに変更
	if err := syscall.Chroot(rootfs); err != nil {
		fmt.Println("[child] chroot error: ", err)
		os.Exit(1)
	}
	// chroot後はカレントディレクトリがルート外になる可能性があるため
	// 明示的に新しいルートディレクトリ(/)に移動
	if err := os.Chdir("/"); err != nil {
		fmt.Println("[child] chdir error: ", err)
		os.Exit(1)
	}

	// 4. procファイルシステムのマウント
	// psコマンド等が動作するようにコンテナ内の/procをマウント
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		fmt.Println("[child] mount proc error: ", err)
		os.Exit(1)
	}

	fmt.Println("[child] exec: ", cmdPath, cmdArgs[1:])

	// 5. ユーザーコマンドの実行
	// 現在のプロセス(child)をユーザー指定のコマンドに置換
	// 指定したコマンドがコンテナ内のPID 1として動作
	if err := syscall.Exec(cmdPath, cmdArgs, os.Environ()); err != nil {
		fmt.Println("[child] exec error: ", err)
		os.Exit(1)
	}
}

// usageAndExit 使用方法を表示してプログラムを終了
func usageAndExit(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	fmt.Fprintln(os.Stderr, "usage: mini-container run <rootfs> <cmd> [args...")
	os.Exit(1)
}
