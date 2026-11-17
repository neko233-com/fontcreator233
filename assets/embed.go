// Package assets embeds the OFL base fonts and license used by fontcreator233.
package assets

import _ "embed"

// BaseSC is the static wght=400 Simplified Chinese base font.
//
//go:embed base/Fontcreator233BaseSC-Regular.ttf
var BaseSC []byte

// BaseKR is the static wght=400 Hangul subset base font.
//
//go:embed base/Fontcreator233BaseKR-Regular.ttf
var BaseKR []byte

// OFL is the SIL Open Font License text covering both base fonts.
//
//go:embed base/OFL.txt
var OFL []byte

// BaseSCHash is the SHA-256 of BaseSC.
const BaseSCHash = "49d4bf9374ea6d2c480a0db25c54c128727743f04803bf383600b09e2881d7e2"

// BaseKRHash is the SHA-256 of BaseKR.
const BaseKRHash = "ae1212653d0b95a09341f33ceb488038a3afb68f8097a98679102c316c50a590"
