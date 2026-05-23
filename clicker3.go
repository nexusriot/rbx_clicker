package main

import (
	"fmt"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procSendInput        = user32.NewProc("SendInput")
	procRegisterHotKey   = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey = user32.NewProc("UnregisterHotKey")
	procGetMessageW      = user32.NewProc("GetMessageW")
	procSetConsoleTitle  = kernel32.NewProc("SetConsoleTitleW")
)

const (
	INPUT_MOUSE          = 0
	MOUSEEVENTF_LEFTDOWN = 0x0002
	MOUSEEVENTF_LEFTUP   = 0x0004
	MOD_CONTROL          = 0x0002
	MOD_SHIFT            = 0x0004
	MOD_NOREPEAT         = 0x4000

	VK_F9        = 0x78
	VK_F10       = 0x79
	VK_OEM_PLUS  = 0xBB
	VK_OEM_MINUS = 0xBD
	VK_ADD       = 0x6B
	VK_SUBTRACT  = 0x6D

	ID_HOTKEY_START = 1
	ID_HOTKEY_STOP  = 2
	ID_HOTKEY_INC1  = 3
	ID_HOTKEY_DEC1  = 4
	ID_HOTKEY_INC2  = 5
	ID_HOTKEY_DEC2  = 6

	MIN_CPS = 1
	MAX_CPS = 100
)

type MOUSEINPUT struct {
	dx, dy      int32
	mouseData   uint32
	dwFlags     uint32
	time        uint32
	dwExtraInfo uintptr
}

type INPUT struct {
	Type uint32
	mi   MOUSEINPUT
}

type MSG struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
}

var (
	clicking int32 // atomic bool
	cps      int32 = 2
)

func click() {
	inputs := []INPUT{
		{Type: INPUT_MOUSE, mi: MOUSEINPUT{dwFlags: MOUSEEVENTF_LEFTDOWN}},
		{Type: INPUT_MOUSE, mi: MOUSEINPUT{dwFlags: MOUSEEVENTF_LEFTUP}},
	}
	_, _, _ = procSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		unsafe.Sizeof(inputs[0]),
	)
}

func registerHotkey(id int, fsModifiers uint, vk uint) error {
	r, _, err := procRegisterHotKey.Call(0, uintptr(id), uintptr(fsModifiers), uintptr(vk))
	if r == 0 {
		return fmt.Errorf("failed to register hotkey %d: %v", id, err)
	}
	return nil
}

func unregisterHotkey(id int) {
	procUnregisterHotKey.Call(0, uintptr(id))
}

func updateTitle() {
	state := "STOPPED"
	if atomic.LoadInt32(&clicking) == 1 {
		state = "CLICKING"
	}
	title := fmt.Sprintf("rbx_clicker  |  CPS: %d  |  %s", atomic.LoadInt32(&cps), state)
	p, _ := syscall.UTF16PtrFromString(title)
	procSetConsoleTitle.Call(uintptr(unsafe.Pointer(p)))
}

func adjustCPS(delta int32) {
	for {
		cur := atomic.LoadInt32(&cps)
		next := cur + delta
		if next < MIN_CPS {
			next = MIN_CPS
		} else if next > MAX_CPS {
			next = MAX_CPS
		}
		if atomic.CompareAndSwapInt32(&cps, cur, next) {
			fmt.Printf("CPS = %d\n", next)
			updateTitle()
			return
		}
	}
}

func main() {
	fmt.Println("Registering hotkeys...")

	// Start/stop — Ctrl+Shift+F9 / Ctrl+Shift+F10
	if err := registerHotkey(ID_HOTKEY_START, MOD_CONTROL|MOD_SHIFT|MOD_NOREPEAT, VK_F9); err != nil {
		fmt.Printf("Failed to register START hotkey: %v\n", err)
	}
	if err := registerHotkey(ID_HOTKEY_STOP, MOD_CONTROL|MOD_SHIFT|MOD_NOREPEAT, VK_F10); err != nil {
		fmt.Printf("Failed to register STOP hotkey: %v\n", err)
	}
	// + / - (main row and numpad), no modifier — global
	if err := registerHotkey(ID_HOTKEY_INC1, MOD_NOREPEAT, VK_OEM_PLUS); err != nil {
		fmt.Printf("Failed to register +: %v\n", err)
	}
	if err := registerHotkey(ID_HOTKEY_DEC1, MOD_NOREPEAT, VK_OEM_MINUS); err != nil {
		fmt.Printf("Failed to register -: %v\n", err)
	}
	if err := registerHotkey(ID_HOTKEY_INC2, MOD_NOREPEAT, VK_ADD); err != nil {
		fmt.Printf("Failed to register numpad +: %v\n", err)
	}
	if err := registerHotkey(ID_HOTKEY_DEC2, MOD_NOREPEAT, VK_SUBTRACT); err != nil {
		fmt.Printf("Failed to register numpad -: %v\n", err)
	}

	defer unregisterHotkey(ID_HOTKEY_START)
	defer unregisterHotkey(ID_HOTKEY_STOP)
	defer unregisterHotkey(ID_HOTKEY_INC1)
	defer unregisterHotkey(ID_HOTKEY_DEC1)
	defer unregisterHotkey(ID_HOTKEY_INC2)
	defer unregisterHotkey(ID_HOTKEY_DEC2)

	updateTitle()

	fmt.Println("Ready.")
	fmt.Println("  Ctrl+Shift+F9   START clicking")
	fmt.Println("  Ctrl+Shift+F10  STOP clicking")
	fmt.Println("  +               increase CPS")
	fmt.Println("  -               decrease CPS")
	fmt.Printf("Current CPS = %d (range %d..%d)\n", atomic.LoadInt32(&cps), MIN_CPS, MAX_CPS)

	go func() {
		for {
			if atomic.LoadInt32(&clicking) == 1 {
				click()
				interval := time.Second / time.Duration(atomic.LoadInt32(&cps))
				time.Sleep(interval)
			} else {
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()

	var msg MSG
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) == -1 {
			fmt.Println("Error in message loop.")
			break
		} else if int32(ret) == 0 {
			fmt.Println("WM_QUIT received. Exiting.")
			break
		}

		switch int(msg.wParam) {
		case ID_HOTKEY_START:
			if atomic.CompareAndSwapInt32(&clicking, 0, 1) {
				fmt.Println("Started clicking.")
				updateTitle()
			}
		case ID_HOTKEY_STOP:
			if atomic.CompareAndSwapInt32(&clicking, 1, 0) {
				fmt.Println("Stopped clicking.")
				updateTitle()
			}
		case ID_HOTKEY_INC1, ID_HOTKEY_INC2:
			adjustCPS(+20)
		case ID_HOTKEY_DEC1, ID_HOTKEY_DEC2:
			adjustCPS(-20)
		}
	}
}
