// +build windows

package winconsole

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

const (
	// Consts for Get/SetConsoleMode function
	// see http://msdn.microsoft.com/en-us/library/windows/desktop/ms683167(v=vs.85).aspx
	ENABLE_ECHO_INPUT      = 0x0004
	ENABLE_INSERT_MODE     = 0x0020
	ENABLE_LINE_INPUT      = 0x0002
	ENABLE_MOUSE_INPUT     = 0x0010
	ENABLE_PROCESSED_INPUT = 0x0001
	ENABLE_QUICK_EDIT_MODE = 0x0040
	ENABLE_WINDOW_INPUT    = 0x0008
	// If parameter is a screen buffer handle, additional values
	ENABLE_PROCESSED_OUTPUT   = 0x0001
	ENABLE_WRAP_AT_EOL_OUTPUT = 0x0002

	//http://msdn.microsoft.com/en-us/library/windows/desktop/ms682088(v=vs.85).aspx#_win32_character_attributes
	FOREGROUND_BLUE       = 1
	FOREGROUND_GREEN      = 2
	FOREGROUND_RED        = 4
	FOREGROUND_INTENSITY  = 8
	FOREGROUND_MASK_SET   = 0x000F
	FOREGROUND_MASK_UNSET = 0xFFF0

	BACKGROUND_BLUE       = 16
	BACKGROUND_GREEN      = 32
	BACKGROUND_RED        = 64
	BACKGROUND_INTENSITY  = 128
	BACKGROUND_MASK_SET   = 0x00F0
	BACKGROUND_MASK_UNSET = 0xFF0F

	COMMON_LVB_REVERSE_VIDEO = 0x4000
	COMMON_LVB_UNDERSCORE    = 0x8000

	// http://man7.org/linux/man-pages/man4/console_codes.4.html
	// ECMA-48 Set Graphics Rendition
	ANSI_ATTR_RESET     = 0
	ANSI_ATTR_BOLD      = 1
	ANSI_ATTR_DIM       = 2
	ANSI_ATTR_UNDERLINE = 4
	ANSI_ATTR_BLINK     = 5
	ANSI_ATTR_REVERSE   = 7
	ANSI_ATTR_INVISIBLE = 8

	ANSI_ATTR_UNDERLINE_OFF = 24
	ANSI_ATTR_BLINK_OFF     = 25
	ANSI_ATTR_REVERSE_OFF   = 27
	ANSI_ATTR_INVISIBLE_OFF = 8

	ANSI_FOREGROUND_BLACK   = 30
	ANSI_FOREGROUND_RED     = 31
	ANSI_FOREGROUND_GREEN   = 32
	ANSI_FOREGROUND_YELLOW  = 33
	ANSI_FOREGROUND_BLUE    = 34
	ANSI_FOREGROUND_MAGENTA = 35
	ANSI_FOREGROUND_CYAN    = 36
	ANSI_FOREGROUND_WHITE   = 37
	ANSI_FOREGROUND_DEFAULT = 39

	ANSI_BACKGROUND_BLACK   = 40
	ANSI_BACKGROUND_RED     = 41
	ANSI_BACKGROUND_GREEN   = 42
	ANSI_BACKGROUND_YELLOW  = 43
	ANSI_BACKGROUND_BLUE    = 44
	ANSI_BACKGROUND_MAGENTA = 45
	ANSI_BACKGROUND_CYAN    = 46
	ANSI_BACKGROUND_WHITE   = 47
	ANSI_BACKGROUND_DEFAULT = 49

	ANSI_MAX_CMD_LENGTH = 256

	MAX_INPUT_BUFFER = 1024
	DEFAULT_WIDTH    = 80
	DEFAULT_HEIGHT   = 24
)

// http://msdn.microsoft.com/en-us/library/windows/desktop/dd375731(v=vs.85).aspx
const (
	VK_PRIOR    = 0x21 // PAGE UP key
	VK_NEXT     = 0x22 // PAGE DOWN key
	VK_END      = 0x23 // END key
	VK_HOME     = 0x24 // HOME key
	VK_LEFT     = 0x25 // LEFT ARROW key
	VK_UP       = 0x26 // UP ARROW key
	VK_RIGHT    = 0x27 //RIGHT ARROW key
	VK_DOWN     = 0x28 //DOWN ARROW key
	VK_SELECT   = 0x29 //SELECT key
	VK_PRINT    = 0x2A //PRINT key
	VK_EXECUTE  = 0x2B //EXECUTE key
	VK_SNAPSHOT = 0x2C //PRINT SCREEN key
	VK_INSERT   = 0x2D //INS key
	VK_DELETE   = 0x2E //DEL key
	VK_HELP     = 0x2F //HELP key
	VK_F1       = 0x70 //F1 key
	VK_F2       = 0x71 //F2 key
	VK_F3       = 0x72 //F3 key
	VK_F4       = 0x73 //F4 key
	VK_F5       = 0x74 //F5 key
	VK_F6       = 0x75 //F6 key
	VK_F7       = 0x76 //F7 key
	VK_F8       = 0x77 //F8 key
	VK_F9       = 0x78 //F9 key
	VK_F10      = 0x79 //F10 key
	VK_F11      = 0x7A //F11 key
	VK_F12      = 0x7B //F12 key
)

var kernel32DLL = syscall.NewLazyDLL("kernel32.dll")

var (
	setConsoleModeProc                = kernel32DLL.NewProc("SetConsoleMode")
	getConsoleScreenBufferInfoProc    = kernel32DLL.NewProc("GetConsoleScreenBufferInfo")
	setConsoleCursorPositionProc      = kernel32DLL.NewProc("SetConsoleCursorPosition")
	setConsoleTextAttributeProc       = kernel32DLL.NewProc("SetConsoleTextAttribute")
	fillConsoleOutputCharacterProc    = kernel32DLL.NewProc("FillConsoleOutputCharacterW")
	writeConsoleOutputProc            = kernel32DLL.NewProc("WriteConsoleOutputW")
	readConsoleInputProc              = kernel32DLL.NewProc("ReadConsoleInputW")
	getNumberOfConsoleInputEventsProc = kernel32DLL.NewProc("GetNumberOfConsoleInputEvents")
	getConsoleCursorInfoProc          = kernel32DLL.NewProc("GetConsoleCursorInfo")
	setConsoleCursorInfoProc          = kernel32DLL.NewProc("SetConsoleCursorInfo")
	setConsoleWindowInfoProc          = kernel32DLL.NewProc("SetConsoleWindowInfo")
	setConsoleScreenBufferSizeProc    = kernel32DLL.NewProc("SetConsoleScreenBufferSize")
)

// types for calling various windows API
// see http://msdn.microsoft.com/en-us/library/windows/desktop/ms682093(v=vs.85).aspx
type (
	SHORT      int16
	SMALL_RECT struct {
		Left   SHORT
		Top    SHORT
		Right  SHORT
		Bottom SHORT
	}

	COORD struct {
		X SHORT
		Y SHORT
	}

	BOOL  int32
	WORD  uint16
	WCHAR uint16
	DWORD uint32

	CONSOLE_SCREEN_BUFFER_INFO struct {
		Size              COORD
		CursorPosition    COORD
		Attributes        WORD
		Window            SMALL_RECT
		MaximumWindowSize COORD
	}

	CONSOLE_CURSOR_INFO struct {
		Size    DWORD
		Visible BOOL
	}

	// http://msdn.microsoft.com/en-us/library/windows/desktop/ms684166(v=vs.85).aspx
	KEY_EVENT_RECORD struct {
		KeyDown         BOOL
		RepeatCount     WORD
		VirtualKeyCode  WORD
		VirtualScanCode WORD
		UnicodeChar     WCHAR
		ControlKeyState DWORD
	}

	INPUT_RECORD struct {
		EventType WORD
		KeyEvent  KEY_EVENT_RECORD
	}

	CHAR_INFO struct {
		UnicodeChar WCHAR
		Attributes  WORD
	}
)

// Implements the TerminalEmulator interface
type WindowsTerminal struct {
	outMutex            sync.Mutex
	inMutex             sync.Mutex
	inputBuffer         chan byte
	screenBufferInfo    *CONSOLE_SCREEN_BUFFER_INFO
	inputEscapeSequence []byte
}

func StdStreams() (stdOut io.Writer, stdErr io.Writer, stdIn io.ReadCloser) {
	handler := &WindowsTerminal{
		inputBuffer:         make(chan byte, MAX_INPUT_BUFFER),
		inputEscapeSequence: []byte(KEY_ESC_CSI),
	}

	// Save current screen buffer info
	handle, err := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	if nil != err {
		panic("This should never happen as it is predefined handle.")
	}
	screenBufferInfo, err := GetConsoleScreenBufferInfo(uintptr(handle))
	if err == nil {
		handler.screenBufferInfo = screenBufferInfo
	}

	// Set the window size
	SetWindowSize(uintptr(handle), DEFAULT_WIDTH, DEFAULT_HEIGHT, DEFAULT_HEIGHT)
	if IsTerminal(os.Stdout.Fd()) {
		stdOut = &terminalWriter{
			wrappedWriter: os.Stdout,
			emulator:      handler,
			command:       make([]byte, 0, ANSI_MAX_CMD_LENGTH),
			fd:            uintptr(handle),
		}
	} else {
		stdOut = os.Stdout

	}
	if IsTerminal(os.Stderr.Fd()) {
		handle, err := syscall.GetStdHandle(syscall.STD_ERROR_HANDLE)
		if nil != err {
			panic("This should never happen as it is predefined handle.")
		}
		stdErr = &terminalWriter{
			wrappedWriter: os.Stderr,
			emulator:      handler,
			command:       make([]byte, 0, ANSI_MAX_CMD_LENGTH),
			fd:            uintptr(handle),
		}
	} else {
		stdErr = os.Stderr
	}
	if IsTerminal(os.Stdin.Fd()) {
		handle, err := syscall.GetStdHandle(syscall.STD_INPUT_HANDLE)
		if nil != err {
			panic("This should never happen as it is predefined handle.")
		}
		stdIn = &terminalReader{
			wrappedReader: os.Stdin,
			emulator:      handler,
			command:       make([]byte, 0, ANSI_MAX_CMD_LENGTH),
			fd:            uintptr(handle),
		}
	} else {
		stdIn = os.Stdin
	}

	return
}

// GetHandleInfo returns file descriptor and bool indicating whether the file is a terminal
func GetHandleInfo(in interface{}) (uintptr, bool) {
	var inFd uintptr
	var isTerminalIn bool
	if tr, ok := in.(*terminalReader); ok {
		if file, ok := tr.wrappedReader.(*os.File); ok {
			inFd = file.Fd()
			isTerminalIn = IsTerminal(inFd)
		}
	}
	return inFd, isTerminalIn
}

// GetConsoleMode gets the console mode for given file descriptor
// http://msdn.microsoft.com/en-us/library/windows/desktop/ms683167(v=vs.85).aspx
func GetConsoleMode(fileDesc uintptr) (uint32, error) {
	var mode uint32
	err := syscall.GetConsoleMode(syscall.Handle(fileDesc), &mode)
	return mode, err
}

// SetConsoleMode sets the console mode for given file descriptor
// http://msdn.microsoft.com/en-us/library/windows/desktop/ms686033(v=vs.85).aspx
func SetConsoleMode(fileDesc uintptr, mode uint32) error {
	r, _, err := setConsoleModeProc.Call(fileDesc, uintptr(mode), 0)
	if r == 0 {
		if err != nil {
			return err
		}
		return syscall.EINVAL
	}
	return nil
}

// SetCursorVisible sets the cursor visbility
// http://msdn.microsoft.com/en-us/library/windows/desktop/ms686019(v=vs.85).aspx
func SetCursorVisible(fileDesc uintptr, isVisible BOOL) (bool, error) {
	var cursorInfo CONSOLE_CURSOR_INFO
	r, _, err := getConsoleCursorInfoProc.Call(uintptr(fileDesc), uintptr(unsafe.Pointer(&cursorInfo)), 0)
	if r == 0 {
		if err != nil {
			return false, err
		}
		return false, syscall.EINVAL
	}
	cursorInfo.Visible = isVisible
	r, _, err = setConsoleCursorInfoProc.Call(uintptr(fileDesc), uintptr(unsafe.Pointer(&cursorInfo)), 0)
	if r == 0 {
		if err != nil {
			return false, err
		}
		return false, syscall.EINVAL
	}
	return true, nil
}

// SetWindowSize sets the size of the console window.
func SetWindowSize(fileDesc uintptr, width, height, max SHORT) (bool, error) {
	window := SMALL_RECT{Left: 0, Top: 0, Right: width - 1, Bottom: height - 1}
	coord := COORD{X: width - 1, Y: max}
	r, _, err := setConsoleWindowInfoProc.Call(uintptr(fileDesc), uintptr(BOOL(1)), uintptr(unsafe.Pointer(&window)))
	if r == 0 {
		if err != nil {
			return false, err
		}
		return false, syscall.EINVAL
	}
	r, _, err = setConsoleScreenBufferSizeProc.Call(uintptr(fileDesc), uintptr(marshal(coord)))
	if r == 0 {
		if err != nil {
			return false, err
		}
		return false, syscall.EINVAL
	}

	return true, nil
}

// GetConsoleScreenBufferInfo retrieves information about the specified console screen buffer.
// http://msdn.microsoft.com/en-us/library/windows/desktop/ms683171(v=vs.85).aspx
func GetConsoleScreenBufferInfo(fileDesc uintptr) (*CONSOLE_SCREEN_BUFFER_INFO, error) {
	var info CONSOLE_SCREEN_BUFFER_INFO
	r, _, err := getConsoleScreenBufferInfoProc.Call(uintptr(fileDesc), uintptr(unsafe.Pointer(&info)), 0)
	if r == 0 {
		if err != nil {
			return nil, err
		}
		return nil, syscall.EINVAL
	}
	return &info, nil
}

// setConsoleTextAttribute sets the attributes of characters written to the
// console screen buffer by the WriteFile or WriteConsole function,
// http://msdn.microsoft.com/en-us/library/windows/desktop/ms686047(v=vs.85).aspx
func setConsoleTextAttribute(fileDesc uintptr, attribute WORD) (bool, error) {
	r, _, err := setConsoleTextAttributeProc.Call(uintptr(fileDesc), uintptr(attribute), 0)
	if r == 0 {
		if err != nil {
			return false, err
		}
		return false, syscall.EINVAL
	}
	return true, nil
}

func writeConsoleOutput(fileDesc uintptr, buffer []CHAR_INFO, bufferSize COORD, bufferCoord COORD, writeRegion *SMALL_RECT) (bool, error) {
	r, _, err := writeConsoleOutputProc.Call(uintptr(fileDesc), uintptr(unsafe.Pointer(&buffer[0])), uintptr(marshal(bufferSize)), uintptr(marshal(bufferCoord)), uintptr(unsafe.Pointer(writeRegion)))
	if r == 0 {
		if err != nil {
			return false, err
		}
		return false, syscall.EINVAL
	}
	return true, nil
}

// http://msdn.microsoft.com/en-us/library/windows/desktop/ms682663(v=vs.85).aspx
func fillConsoleOutputCharacter(fileDesc uintptr, fillChar byte, length uint32, writeCord COORD) (bool, error) {
	out := int64(0)
	r, _, err := fillConsoleOutputCharacterProc.Call(uintptr(fileDesc), uintptr(fillChar), uintptr(length), uintptr(marshal(writeCord)), uintptr(unsafe.Pointer(&out)))
	// If the function succeeds, the return value is nonzero.
	if r == 0 {
		if err != nil {
			return false, err
		}
		return false, syscall.EINVAL
	}
	return true, nil
}

// Gets the number of space characters to write for "clearing" the section of terminal
func getNumberOfChars(fromCoord COORD, toCoord COORD, screenSize COORD) uint32 {
	// must be valid cursor position
	if fromCoord.X < 0 || fromCoord.Y < 0 || toCoord.X < 0 || toCoord.Y < 0 {
		return 0
	}
	if fromCoord.X >= screenSize.X || fromCoord.Y >= screenSize.Y || toCoord.X >= screenSize.X || toCoord.Y >= screenSize.Y {
		return 0
	}
	// can't be backwards
	if fromCoord.Y > toCoord.Y {
		return 0
	}
	// same line
	if fromCoord.Y == toCoord.Y {
		return uint32(toCoord.X-fromCoord.X) + 1
	}
	// spans more than one line
	if fromCoord.Y < toCoord.Y {
		// from start till end of line for first line +  from start of line till end
		retValue := uint32(screenSize.X-fromCoord.X) + uint32(toCoord.X) + 1
		// don't count first and last line
		linesBetween := toCoord.Y - fromCoord.Y - 1
		if linesBetween > 0 {
			retValue = retValue + uint32(linesBetween*screenSize.X)
		}
		return retValue
	}
	return 0
}

func clearDisplayRect(fileDesc uintptr, fillChar rune, attributes WORD, fromCoord COORD, toCoord COORD, windowSize COORD) (bool, uint32, error) {
	var writeRegion SMALL_RECT
	writeRegion.Top = fromCoord.Y
	writeRegion.Left = fromCoord.X
	writeRegion.Right = toCoord.X
	writeRegion.Bottom = toCoord.Y

	// allocate and initialize buffer
	width := toCoord.X - fromCoord.X + 1
	height := toCoord.Y - fromCoord.Y + 1
	size := width * height
	if size > 0 {
		buffer := make([]CHAR_INFO, size)
		for i := 0; i < len(buffer); i++ {
			buffer[i].UnicodeChar = WCHAR(fillChar)
			buffer[i].Attributes = attributes
		}

		// Write to buffer
		r, err := writeConsoleOutput(fileDesc, buffer, windowSize, COORD{X: 0, Y: 0}, &writeRegion)
		if !r {
			if err != nil {
				return false, 0, err
			}
			return false, 0, syscall.EINVAL
		}
	}
	return true, uint32(size), nil
}

func clearDisplayRange(fileDesc uintptr, fillChar rune, attributes WORD, fromCoord COORD, toCoord COORD, windowSize COORD) (bool, uint32, error) {
	nw := uint32(0)
	// start and end on same line
	if fromCoord.Y == toCoord.Y {
		r, charWritten, err := clearDisplayRect(fileDesc, fillChar, attributes, fromCoord, toCoord, windowSize)
		if !r {
			if err != nil {
				return false, charWritten, err
			}
			return false, charWritten, syscall.EINVAL
		}
		return true, charWritten, nil
	}
	// TODO(azlinux): if full screen, optimize

	// spans more than one line
	if fromCoord.Y < toCoord.Y {
		// from start position till end of line for first line
		r, n, err := clearDisplayRect(fileDesc, fillChar, attributes, fromCoord, COORD{X: windowSize.X - 1, Y: fromCoord.Y}, windowSize)
		if !r {
			if err != nil {
				return false, nw, err
			}
			return false, nw, syscall.EINVAL
		}
		nw += n
		// lines between
		linesBetween := toCoord.Y - fromCoord.Y - 1
		if linesBetween > 0 {
			r, n, err = clearDisplayRect(fileDesc, fillChar, attributes, COORD{X: 0, Y: fromCoord.Y + 1}, COORD{X: windowSize.X - 1, Y: toCoord.Y - 1}, windowSize)
			if !r {
				if err != nil {
					return false, nw, err
				}
				return false, nw, syscall.EINVAL
			}
			nw += n
		}
		// lines at end
		r, n, err = clearDisplayRect(fileDesc, fillChar, attributes, COORD{X: 0, Y: toCoord.Y}, toCoord, windowSize)
		if !r {
			if err != nil {
				return false, nw, err
			}
			return false, nw, syscall.EINVAL
		}
		nw += n
	}
	return true, nw, nil
}

// setConsoleCursorPosition sets the console cursor position
// Note The X and Y are zero based
// If relative is true then the new position is relative to current one
func setConsoleCursorPosition(fileDesc uintptr, isRelative bool, column int16, line int16) (bool, error) {
	screenBufferInfo, err := GetConsoleScreenBufferInfo(fileDesc)
	if err == nil {
		var position COORD
		if isRelative {
			position.X = screenBufferInfo.CursorPosition.X + SHORT(column)
			position.Y = screenBufferInfo.CursorPosition.Y + SHORT(line)
		} else {
			position.X = SHORT(column)
			position.Y = SHORT(line)
		}

		//convert
		bits := marshal(position)
		r, _, err := setConsoleCursorPositionProc.Call(uintptr(fileDesc), uintptr(bits), 0)
		if r == 0 {
			if err != nil {
				return false, err
			}
			return false, syscall.EINVAL
		}
		return true, nil
	}
	return false, err
}

// http://msdn.microsoft.com/en-us/library/windows/desktop/ms683207(v=vs.85).aspx
func getNumberOfConsoleInputEvents(fileDesc uintptr) (uint16, error) {
	var n WORD
	r, _, err := getNumberOfConsoleInputEventsProc.Call(uintptr(fileDesc), uintptr(unsafe.Pointer(&n)))
	//If the function succeeds, the return value is nonzero
	if r != 0 {
		return uint16(n), nil
	}
	return 0, err
}

//http://msdn.microsoft.com/en-us/library/windows/desktop/ms684961(v=vs.85).aspx
func readConsoleInputKey(fileDesc uintptr, inputBuffer []INPUT_RECORD) (int, error) {
	var nr WORD
	r, _, err := readConsoleInputProc.Call(uintptr(fileDesc), uintptr(unsafe.Pointer(&inputBuffer[0])), uintptr(WORD(len(inputBuffer))), uintptr(unsafe.Pointer(&nr)))
	//If the function succeeds, the return value is nonzero.
	if r != 0 {
		return int(nr), nil
	}
	return int(0), err
}

func getWindowsTextAttributeForAnsiValue(originalFlag WORD, defaultValue WORD, ansiValue int16) (WORD, error) {
	flag := WORD(originalFlag)
	if flag == 0 {
		flag = defaultValue
	}
	switch ansiValue {
	case ANSI_ATTR_RESET:
		flag &^= COMMON_LVB_UNDERSCORE
		flag &^= BACKGROUND_INTENSITY
		flag = flag | FOREGROUND_INTENSITY
	case ANSI_ATTR_INVISIBLE:
		// TODO: how do you reset reverse?
	case ANSI_ATTR_UNDERLINE:
		flag = flag | COMMON_LVB_UNDERSCORE
	case ANSI_ATTR_BLINK:
		// seems like background intenisty is blink
		flag = flag | BACKGROUND_INTENSITY
	case ANSI_ATTR_UNDERLINE_OFF:
		flag &^= COMMON_LVB_UNDERSCORE
	case ANSI_ATTR_BLINK_OFF:
		// seems like background intenisty is blink
		flag &^= BACKGROUND_INTENSITY
	case ANSI_ATTR_BOLD:
		flag = flag | FOREGROUND_INTENSITY
	case ANSI_ATTR_DIM:
		flag &^= FOREGROUND_INTENSITY
	case ANSI_ATTR_REVERSE, ANSI_ATTR_REVERSE_OFF:
		// swap forground and background bits
		foreground := flag & FOREGROUND_MASK_SET
		background := flag & BACKGROUND_MASK_SET
		flag = (flag & BACKGROUND_MASK_UNSET & FOREGROUND_MASK_UNSET) | (foreground << 4) | (background >> 4)

	// FOREGROUND
	case ANSI_FOREGROUND_DEFAULT:
		flag = (flag & FOREGROUND_MASK_UNSET) | (defaultValue & FOREGROUND_MASK_SET)
	case ANSI_FOREGROUND_BLACK:
		flag = flag ^ (FOREGROUND_RED | FOREGROUND_GREEN | FOREGROUND_BLUE)
	case ANSI_FOREGROUND_RED:
		flag = (flag & FOREGROUND_MASK_UNSET) | FOREGROUND_RED
	case ANSI_FOREGROUND_GREEN:
		flag = (flag & FOREGROUND_MASK_UNSET) | FOREGROUND_GREEN
	case ANSI_FOREGROUND_YELLOW:
		flag = (flag & FOREGROUND_MASK_UNSET) | FOREGROUND_RED | FOREGROUND_GREEN
	case ANSI_FOREGROUND_BLUE:
		flag = (flag & FOREGROUND_MASK_UNSET) | FOREGROUND_BLUE
	case ANSI_FOREGROUND_MAGENTA:
		flag = (flag & FOREGROUND_MASK_UNSET) | FOREGROUND_RED | FOREGROUND_BLUE
	case ANSI_FOREGROUND_CYAN:
		flag = (flag & FOREGROUND_MASK_UNSET) | FOREGROUND_GREEN | FOREGROUND_BLUE
	case ANSI_FOREGROUND_WHITE:
		flag = (flag & FOREGROUND_MASK_UNSET) | FOREGROUND_RED | FOREGROUND_GREEN | FOREGROUND_BLUE

	// Background
	case ANSI_BACKGROUND_DEFAULT:
		// Black with no intensity
		flag = (flag & BACKGROUND_MASK_UNSET) | (defaultValue & BACKGROUND_MASK_SET)
	case ANSI_BACKGROUND_BLACK:
		flag = (flag & BACKGROUND_MASK_UNSET)
	case ANSI_BACKGROUND_RED:
		flag = (flag & BACKGROUND_MASK_UNSET) | BACKGROUND_RED
	case ANSI_BACKGROUND_GREEN:
		flag = (flag & BACKGROUND_MASK_UNSET) | BACKGROUND_GREEN
	case ANSI_BACKGROUND_YELLOW:
		flag = (flag & BACKGROUND_MASK_UNSET) | BACKGROUND_RED | BACKGROUND_GREEN
	case ANSI_BACKGROUND_BLUE:
		flag = (flag & BACKGROUND_MASK_UNSET) | BACKGROUND_BLUE
	case ANSI_BACKGROUND_MAGENTA:
		flag = (flag & BACKGROUND_MASK_UNSET) | BACKGROUND_RED | BACKGROUND_BLUE
	case ANSI_BACKGROUND_CYAN:
		flag = (flag & BACKGROUND_MASK_UNSET) | BACKGROUND_GREEN | BACKGROUND_BLUE
	case ANSI_BACKGROUND_WHITE:
		flag = (flag & BACKGROUND_MASK_UNSET) | BACKGROUND_RED | BACKGROUND_GREEN | BACKGROUND_BLUE
	default:

	}
	return flag, nil
}

// HandleOutputCommand interpretes the Ansi commands and then makes appropriate Win32 calls
func (term *WindowsTerminal) HandleOutputCommand(fd uintptr, command []byte) (n int, err error) {
	// console settings changes need to happen in atomic way
	term.outMutex.Lock()
	defer term.outMutex.Unlock()

	r := false
	// Parse the command
	parsedCommand := parseAnsiCommand(command)

	// use appropriate handle
	handle := syscall.Handle(fd)

	switch parsedCommand.Command {
	case "m":
		// [Value;...;Valuem
		// Set Graphics Mode:
		// Calls the graphics functions specified by the following values.
		// These specified functions remain active until the next occurrence of this escape sequence.
		// Graphics mode changes the colors and attributes of text (such as bold and underline) displayed on the screen.
		screenBufferInfo, err := GetConsoleScreenBufferInfo(uintptr(handle))
		if err != nil {
			return len(command), err
		}
		flag := screenBufferInfo.Attributes
		for _, e := range parsedCommand.Parameters {
			value, _ := strconv.ParseInt(e, 10, 16) // base 10, 16 bit
			if value == ANSI_ATTR_RESET {
				flag = term.screenBufferInfo.Attributes // reset
			} else {
				flag, err = getWindowsTextAttributeForAnsiValue(flag, term.screenBufferInfo.Attributes, int16(value))
				if nil != err {
					return len(command), err
				}
			}
		}
		r, err = setConsoleTextAttribute(uintptr(handle), flag)
		if !r {
			return len(command), err
		}
	case "H", "f":
		// [line;columnH
		// [line;columnf
		// Moves the cursor to the specified position (coordinates).
		// If you do not specify a position, the cursor moves to the home position at the upper-left corner of the screen (line 0, column 0).
		line, err := parseInt16OrDefault(parsedCommand.getParam(0), 1)
		if err != nil {
			return len(command), err
		}
		column, err := parseInt16OrDefault(parsedCommand.getParam(1), 1)
		if err != nil {
			return len(command), err
		}
		// The numbers are not 0 based, but 1 based
		r, err = setConsoleCursorPosition(uintptr(handle), false, int16(column-1), int16(line-1))
		if !r {
			return len(command), err
		}

	case "A":
		// [valueA
		// Moves the cursor up by the specified number of lines without changing columns.
		// If the cursor is already on the top line, ignores this sequence.
		value, err := parseInt16OrDefault(parsedCommand.getParam(0), 1)
		if err != nil {
			return len(command), err
		}
		r, err = setConsoleCursorPosition(uintptr(handle), true, 0, -1*value)
		if !r {
			return len(command), err
		}
	case "B":
		// [valueB
		// Moves the cursor down by the specified number of lines without changing columns.
		// If the cursor is already on the bottom line, ignores this sequence.
		value, err := parseInt16OrDefault(parsedCommand.getParam(0), 1)
		if err != nil {
			return len(command), err
		}
		r, err = setConsoleCursorPosition(uintptr(handle), true, 0, value)
		if !r {
			return len(command), err
		}
	case "C":
		// [valueC
		// Moves the cursor forward by the specified number of columns without changing lines.
		// If the cursor is already in the rightmost column, ignores this sequence.
		value, err := parseInt16OrDefault(parsedCommand.getParam(0), 1)
		if err != nil {
			return len(command), err
		}
		r, err = setConsoleCursorPosition(uintptr(handle), true, int16(value), 0)
		if !r {
			return len(command), err
		}
	case "D":
		// [valueD
		// Moves the cursor back by the specified number of columns without changing lines.
		// If the cursor is already in the leftmost column, ignores this sequence.
		value, err := parseInt16OrDefault(parsedCommand.getParam(0), 1)
		if err != nil {
			return len(command), err
		}
		r, err = setConsoleCursorPosition(uintptr(handle), true, int16(-1*value), 0)
		if !r {
			return len(command), err
		}
	case "J":
		// [J   Erases from the cursor to the end of the screen, including the cursor position.
		// [1J  Erases from the beginning of the screen to the cursor, including the cursor position.
		// [2J  Erases the complete display. The cursor does not move.
		// Clears the screen and moves the cursor to the home position (line 0, column 0).
		value, err := parseInt16OrDefault(parsedCommand.getParam(0), 0)
		if err != nil {
			return len(command), err
		}
		var start COORD
		var cursor COORD
		var end COORD
		screenBufferInfo, err := GetConsoleScreenBufferInfo(uintptr(handle))
		if err == nil {
			switch value {
			case 0:
				start = screenBufferInfo.CursorPosition
				// end of the screen
				end.X = screenBufferInfo.MaximumWindowSize.X - 1
				end.Y = screenBufferInfo.MaximumWindowSize.Y - 1
				// cursor
				cursor = screenBufferInfo.CursorPosition
			case 1:

				// start of the screen
				start.X = 0
				start.Y = 0
				// end of the screen
				end = screenBufferInfo.CursorPosition
				// cursor
				cursor = screenBufferInfo.CursorPosition
			case 2:
				// start of the screen
				start.X = 0
				start.Y = 0
				// end of the screen
				end.X = screenBufferInfo.MaximumWindowSize.X - 1
				end.Y = screenBufferInfo.MaximumWindowSize.Y - 1
				// cursor
				cursor.X = 0
				cursor.Y = 0
			}
			r, _, err = clearDisplayRange(uintptr(handle), ' ', term.screenBufferInfo.Attributes, start, end, screenBufferInfo.MaximumWindowSize)
			if !r {
				return len(command), err
			}
			// remember the the cursor position is 1 based
			r, err = setConsoleCursorPosition(uintptr(handle), false, int16(cursor.X), int16(cursor.Y))
			if !r {
				return len(command), err
			}
		}
	case "K":
		// [K
		// Clears all characters from the cursor position to the end of the line (including the character at the cursor position).
		// [K  Erases from the cursor to the end of the line, including the cursor position.
		// [1K  Erases from the beginning of the line to the cursor, including the cursor position.
		// [2K  Erases the complete line.
		value, err := parseInt16OrDefault(parsedCommand.getParam(0), 0)
		var start COORD
		var cursor COORD
		var end COORD
		screenBufferInfo, err := GetConsoleScreenBufferInfo(uintptr(handle))
		if err == nil {
			switch value {
			case 0:
				// start is where cursor is
				start = screenBufferInfo.CursorPosition
				// end of line
				end.X = screenBufferInfo.MaximumWindowSize.X - 1
				end.Y = screenBufferInfo.CursorPosition.Y
				// cursor remains the same
				cursor = screenBufferInfo.CursorPosition

			case 1:
				// beginning of line
				start.X = 0
				start.Y = screenBufferInfo.CursorPosition.Y
				// until cursor
				end = screenBufferInfo.CursorPosition
				// cursor remains the same
				cursor = screenBufferInfo.CursorPosition
			case 2:
				// start of the line
				start.X = 0
				start.Y = screenBufferInfo.MaximumWindowSize.Y - 1
				// end of the line
				end.X = screenBufferInfo.MaximumWindowSize.X - 1
				end.Y = screenBufferInfo.MaximumWindowSize.Y - 1
				// cursor
				cursor.X = 0
				cursor.Y = screenBufferInfo.MaximumWindowSize.Y - 1
			}
			r, _, err = clearDisplayRange(uintptr(handle), ' ', term.screenBufferInfo.Attributes, start, end, screenBufferInfo.MaximumWindowSize)
			if !r {
				return len(command), err
			}
			// remember the the cursor position is 1 based
			r, err = setConsoleCursorPosition(uintptr(handle), false, int16(cursor.X), int16(cursor.Y))
			if !r {
				return len(command), err
			}
		}

	case "l":
		for _, value := range parsedCommand.Parameters {
			switch value {
			case "?25", "25":
				SetCursorVisible(uintptr(handle), BOOL(0))
			case "?1049", "1049":
				// TODO (azlinux):  Restore terminal
			case "?1", "1":
				// If the DECCKM function is reset, then the arrow keys send ANSI cursor sequences to the host.
				term.inputEscapeSequence = []byte(KEY_ESC_CSI)
			default:
			}
		}
	case "h":
		for _, value := range parsedCommand.Parameters {
			switch value {
			case "?25", "25":
				SetCursorVisible(uintptr(handle), BOOL(1))
			case "?1049", "1049":
				// TODO (azlinux): Save terminal
			case "?1", "1":
				// If the DECCKM function is set, then the arrow keys send application sequences to the host.
				// DECCKM (default off): When set, the cursor keys send an ESC O prefix, rather than ESC [.
				term.inputEscapeSequence = []byte(KEY_ESC_O)
			default:
			}
		}

	case "]":
	/*
		TODO (azlinux):
			Linux Console Private CSI Sequences

		       The following sequences are neither ECMA-48 nor native VT102.  They are
		       native  to the Linux console driver.  Colors are in SGR parameters: 0 =
		       black, 1 = red, 2 = green, 3 = brown, 4 = blue, 5 = magenta, 6 =  cyan,
		       7 = white.

		       ESC [ 1 ; n ]       Set color n as the underline color
		       ESC [ 2 ; n ]       Set color n as the dim color
		       ESC [ 8 ]           Make the current color pair the default attributes.
		       ESC [ 9 ; n ]       Set screen blank timeout to n minutes.
		       ESC [ 10 ; n ]      Set bell frequency in Hz.
		       ESC [ 11 ; n ]      Set bell duration in msec.
		       ESC [ 12 ; n ]      Bring specified console to the front.
		       ESC [ 13 ]          Unblank the screen.
		       ESC [ 14 ; n ]      Set the VESA powerdown interval in minutes.

	*/
	default:
	}
	return len(command), nil
}

// WriteChars writes the bytes to given writer.
func (term *WindowsTerminal) WriteChars(fd uintptr, w io.Writer, p []byte) (n int, err error) {
	return w.Write(p)
}

const (
	CAPSLOCK_ON        = 0x0080 //The CAPS LOCK light is on.
	ENHANCED_KEY       = 0x0100 //The key is enhanced.
	LEFT_ALT_PRESSED   = 0x0002 //The left ALT key is pressed.
	LEFT_CTRL_PRESSED  = 0x0008 //The left CTRL key is pressed.
	NUMLOCK_ON         = 0x0020 //The NUM LOCK light is on.
	RIGHT_ALT_PRESSED  = 0x0001 //The right ALT key is pressed.
	RIGHT_CTRL_PRESSED = 0x0004 //The right CTRL key is pressed.
	SCROLLLOCK_ON      = 0x0040 //The SCROLL LOCK light is on.
	SHIFT_PRESSED      = 0x0010 // The SHIFT key is pressed.
)

const (
	KEY_CONTROL_PARAM_2 = ";2"
	KEY_CONTROL_PARAM_3 = ";3"
	KEY_CONTROL_PARAM_4 = ";4"
	KEY_CONTROL_PARAM_5 = ";5"
	KEY_CONTROL_PARAM_6 = ";6"
	KEY_CONTROL_PARAM_7 = ";7"
	KEY_CONTROL_PARAM_8 = ";8"
	KEY_ESC_CSI         = "\x1B["
	KEY_ESC_N           = "\x1BN"
	KEY_ESC_O           = "\x1BO"
)

var keyMapPrefix = map[WORD]string{
	VK_UP:     "\x1B[%sA",
	VK_DOWN:   "\x1B[%sB",
	VK_RIGHT:  "\x1B[%sC",
	VK_LEFT:   "\x1B[%sD",
	VK_HOME:   "\x1B[1%s~", // showkey shows ^[[1
	VK_END:    "\x1B[4%s~", // showkey shows ^[[4
	VK_INSERT: "\x1B[2%s~",
	VK_DELETE: "\x1B[3%s~",
	VK_PRIOR:  "\x1B[5%s~",
	VK_NEXT:   "\x1B[6%s~",
	VK_F1:     "",
	VK_F2:     "",
	VK_F3:     "\x1B[13%s~",
	VK_F4:     "\x1B[14%s~",
	VK_F5:     "\x1B[15%s~",
	VK_F6:     "\x1B[17%s~",
	VK_F7:     "\x1B[18%s~",
	VK_F8:     "\x1B[19%s~",
	VK_F9:     "\x1B[20%s~",
	VK_F10:    "\x1B[21%s~",
	VK_F11:    "\x1B[23%s~",
	VK_F12:    "\x1B[24%s~",
}

var arrowKeyMapPrefix = map[WORD]string{
	VK_UP:    "%s%sA",
	VK_DOWN:  "%s%sB",
	VK_RIGHT: "%s%sC",
	VK_LEFT:  "%s%sD",
}

func getControlStateParameter(shift, alt, control, meta bool) string {
	if shift && alt && control {
		return KEY_CONTROL_PARAM_8
	}
	if alt && control {
		return KEY_CONTROL_PARAM_7
	}
	if shift && control {
		return KEY_CONTROL_PARAM_6
	}
	if control {
		return KEY_CONTROL_PARAM_5
	}
	if shift && alt {
		return KEY_CONTROL_PARAM_4
	}
	if alt {
		return KEY_CONTROL_PARAM_3
	}
	if shift {
		return KEY_CONTROL_PARAM_2
	}
	return ""
}

func getControlKeys(controlState DWORD) (shift, alt, control bool) {
	shift = 0 != (controlState & SHIFT_PRESSED)
	alt = 0 != (controlState & (LEFT_ALT_PRESSED | RIGHT_ALT_PRESSED))
	control = 0 != (controlState & (LEFT_CTRL_PRESSED | RIGHT_CTRL_PRESSED))
	return shift, alt, control
}

func charSequenceForKeys(key WORD, controlState DWORD, escapeSequence []byte) string {
	i, ok := arrowKeyMapPrefix[key]
	if ok {
		shift, alt, control := getControlKeys(controlState)
		modifier := getControlStateParameter(shift, alt, control, false)
		return fmt.Sprintf(i, escapeSequence, modifier)
	}

	i, ok = keyMapPrefix[key]
	if ok {
		shift, alt, control := getControlKeys(controlState)
		modifier := getControlStateParameter(shift, alt, control, false)
		return fmt.Sprintf(i, modifier)
	}

	return ""
}

// mapKeystokeToTerminalString maps the given input event record to string
func mapKeystokeToTerminalString(keyEvent *KEY_EVENT_RECORD, escapeSequence []byte) string {
	_, alt, control := getControlKeys(keyEvent.ControlKeyState)
	if keyEvent.UnicodeChar == 0 {
		return charSequenceForKeys(keyEvent.VirtualKeyCode, keyEvent.ControlKeyState, escapeSequence)
	}
	if control {
		// TODO(azlinux): Implement following control sequences
		// <Ctrl>-D  Signals the end of input from the keyboard; also exits current shell.
		// <Ctrl>-H  Deletes the first character to the left of the cursor. Also called the ERASE key.
		// <Ctrl>-Q  Restarts printing after it has been stopped with <Ctrl>-s.
		// <Ctrl>-S  Suspends printing on the screen (does not stop the program).
		// <Ctrl>-U  Deletes all characters on the current line. Also called the KILL key.
		// <Ctrl>-E  Quits current command and creates a core

	}
	// <Alt>+Key generates ESC N Key
	if !control && alt {
		return KEY_ESC_N + strings.ToLower(string(keyEvent.UnicodeChar))
	}
	return string(keyEvent.UnicodeChar)
}

// getAvailableInputEvents polls the console for availble events
// The function does not return until at least one input record has been read.
func getAvailableInputEvents(fd uintptr) (inputEvents []INPUT_RECORD, err error) {
	handle := syscall.Handle(fd)
	if nil != err {
		return nil, err
	}
	for {
		// Read number of console events available
		tempBuffer := make([]INPUT_RECORD, MAX_INPUT_BUFFER)
		nr, err := readConsoleInputKey(uintptr(handle), tempBuffer)
		if nr == 0 {
			return nil, err
		}
		if 0 < nr {
			retValue := make([]INPUT_RECORD, nr)
			for i := 0; i < nr; i++ {
				retValue[i] = tempBuffer[i]
			}
			return retValue, nil
		}
	}
}

// getTranslatedKeyCodes converts the input events into the string of characters
// The ansi escape sequence are used to map key strokes to the strings
func getTranslatedKeyCodes(inputEvents []INPUT_RECORD, escapeSequence []byte) string {
	var buf bytes.Buffer
	for i := 0; i < len(inputEvents); i++ {
		input := inputEvents[i]
		if input.EventType == KEY_EVENT && input.KeyEvent.KeyDown != 0 {
			keyString := mapKeystokeToTerminalString(&input.KeyEvent, escapeSequence)
			buf.WriteString(keyString)
		}
	}
	return buf.String()
}

// ReadChars reads the characters from the given reader
func (term *WindowsTerminal) ReadChars(fd uintptr, w io.Reader, p []byte) (n int, err error) {
	n = 0
	for n < len(p) {
		select {
		case b := <-term.inputBuffer:
			p[n] = b
			n++
		default:
			// Read at least one byte read
			if n > 0 {
				return n, nil
			}
			inputEvents, _ := getAvailableInputEvents(fd)
			if inputEvents != nil {
				if len(inputEvents) == 0 && nil != err {
					return n, err
				}
				if len(inputEvents) != 0 {
					keyCodes := getTranslatedKeyCodes(inputEvents, term.inputEscapeSequence)
					for _, b := range []byte(keyCodes) {
						term.inputBuffer <- b
					}
				}
			}
		}
	}
	return n, nil
}

// HandleInputSequence interprets the input sequence command
func (term *WindowsTerminal) HandleInputSequence(fd uintptr, command []byte) (n int, err error) {
	return 0, nil
}

func marshal(c COORD) uint32 {
	// works only on intel-endian machines
	return uint32(uint32(uint16(c.Y))<<16 | uint32(uint16(c.X)))
}

// IsTerminal returns true if the given file descriptor is a terminal.
func IsTerminal(fd uintptr) bool {
	_, e := GetConsoleMode(fd)
	return e == nil
}
