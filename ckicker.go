package main

import (
    "fmt"
    "syscall"
    "time"
    "unsafe"
)

var (
    user32             = syscall.NewLazyDLL("user32.dll")
    procSendInput      = user32.NewProc("SendInput")
    procRegisterHotKey = user32.NewProc("RegisterHotKey")
    procUnregisterHotKey = user32.NewProc("UnregisterHotKey")
    procGetMessageW    = user32.NewProc("GetMessageW")
)

const (
    INPUT_MOUSE         = 0
    MOUSEEVENTF_LEFTDOWN = 0x0002
    MOUSEEVENTF_LEFTUP   = 0x0004
    MOD_NOREPEAT         = 0x4000

    ID_HOTKEY_START = 1
    ID_HOTKEY_STOP  = 2
)

type MOUSEINPUT struct {
    dx, dy       int32
    mouseData    uint32
    dwFlags      uint32
    time         uint32
    dwExtraInfo  uintptr
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

var clicking = false

func click() {
    inputs := []INPUT{
        {
            Type: INPUT_MOUSE,
            mi: MOUSEINPUT{dwFlags: MOUSEEVENTF_LEFTDOWN},
        },
        {
            Type: INPUT_MOUSE,
            mi: MOUSEINPUT{dwFlags: MOUSEEVENTF_LEFTUP},
        },
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
        return fmt.Errorf("failed to register hotkey: %v", err)
    }
    return nil
}

func unregisterHotkey(id int) {
    procUnregisterHotKey.Call(0, uintptr(id))
}

func main() {
    // Register F6 (start) and F7 (stop)
    if err := registerHotkey(ID_HOTKEY_START, MOD_NOREPEAT, 0x75); err != nil { // F6
        panic(err)
    }
    if err := registerHotkey(ID_HOTKEY_STOP, MOD_NOREPEAT, 0x76); err != nil { // F7
        panic(err)
    }
    defer unregisterHotkey(ID_HOTKEY_START)
    defer unregisterHotkey(ID_HOTKEY_STOP)

    fmt.Println("Press F6 to start clicking, F7 to stop.")

    go func() {
        for {
            if clicking {
                click()
                time.Sleep(500 * time.Millisecond)
            } else {
                time.Sleep(100 * time.Millisecond)
            }
        }
    }()

    // Message loop to catch hotkey events
    var msg MSG
    for {
        _, _, _ = procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)

        switch msg.wParam {
        case ID_HOTKEY_START:
            clicking = true
            fmt.Println("Started clicking.")
        case ID_HOTKEY_STOP:
            clicking = false
            fmt.Println("Stopped clicking.")
        }
    }
}

