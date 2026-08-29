package screen

import "strings"

const secretMaskRune = '•'

// MaskSecret returns a display-only mask containing one bullet per supported
// display cell in secret. It never returns any part of secret and does not
// mutate the input string. The output may reveal an approximate display width;
// masking is presentation, not a security boundary.
func MaskSecret(secret string) string {
	return strings.Repeat(string(secretMaskRune), TextWidth(secret))
}

// DrawMaskedText places a display-only secret mask in the frame. The secret is
// converted directly to its mask and is never stored in a Cell.
func (f *Frame) DrawMaskedText(row, column int, style Style, secret string) {
	if f == nil {
		return
	}
	f.DrawText(row, column, style, MaskSecret(secret))
}

// WriteMasked writes only the display mask for secret using style. The secret
// is not passed to the terminal writer or included in any error text.
func (t *Terminal) WriteMasked(style Style, secret string) error {
	return t.WriteStyled(style, MaskSecret(secret))
}
