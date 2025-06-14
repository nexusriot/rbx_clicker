package main

import (
    "fmt"
    "syscall"
    "time"
    "unsafe"
)

var (
    user32               = syscall.NewLazyDLL("user32.dll")
    procSendInput        = user32.NewProc("SendInput")
    procRegisterHotKey   = user32.NewProc("RegisterHotKey")
    procUnregisterHotKey = user32.NewProc("UnregisterHotKey")
    procGetMessageW      = user32.NewProc("GetMessageW")
)

const (
    INPUT_MOUSE          = 0
    MOUSEEVENTF_LEFTDOWN = 0x0002
    MOUSEEVENTF_LEFTUP   = 0x0004
    MOD_CONTROL          = 0x0002
    MOD_SHIFT            = 0x0004
    MOD_NOREPEAT         = 0x4000

    ID_HOTKEY_START      = 1
    ID_HOTKEY_STOP       = 2
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

var clicking = false

func click() {
    inputs := []INPUT{
        {
            Type: INPUT_MOUSE,
            mi:   MOUSEINPUT{dwFlags: MOUSEEVENTF_LEFTDOWN},
        },
        {
            Type: INPUT_MOUSE,
            mi:   MOUSEINPUT{dwFlags: MOUSEEVENTF_LEFTUP},
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
        return fmt.Errorf("failed to register hotkey %d: %v", id, err)
    }
    return nil
}

func unregisterHotkey(id int) {
    procUnregisterHotKey.Call(0, uintptr(id))
}

func main() {
    fmt.Println("Registering hotkeys...")

    // Ctrl+Shift+F9 = Start | Ctrl+Shift+F10 = Stop
    if err := registerHotkey(ID_HOTKEY_START, MOD_CONTROL|MOD_SHIFT|MOD_NOREPEAT, 0x78); err != nil {
        fmt.Printf("Failed to register START hotkey: %v\n", err)
    }
    if err := registerHotkey(ID_HOTKEY_STOP, MOD_CONTROL|MOD_SHIFT|MOD_NOREPEAT, 0x79); err != nil {
        fmt.Printf("Failed to register STOP hotkey: %v\n", err)
    }

    defer unregisterHotkey(ID_HOTKEY_START)
    defer unregisterHotkey(ID_HOTKEY_STOP)

    fmt.Println("Ready.")
    fmt.Println("  Press Ctrl+Shift+F9 to START clicking")
    fmt.Println("  Press Ctrl+Shift+F10 to STOP clicking")

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
            if !clicking {
                clicking = true
                fmt.Println("Started clicking.")
            }
        case ID_HOTKEY_STOP:
            if clicking {
                clicking = false
                fmt.Println("Stopped clicking.")
            }
        }
    }
}
