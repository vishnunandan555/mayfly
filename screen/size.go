package screen

import "errors"

// ErrTerminalSizeUnsupported is returned when the terminal size cannot be
// queried through the operating system interface available to this build.
// Applications may provide an explicit Size when using NewApplication.
var ErrTerminalSizeUnsupported = errors.New("screen: terminal size unsupported")
