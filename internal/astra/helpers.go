package astra

import (
	"fmt"
	"strings"
	"unicode"
)

// ██╗██████╗
// ██║██╔══██╗
// ██║██║  ██║
// ██║██║  ██║
// ██║██████╔╝
// ╚═╝╚═════╝

func GenerateStreamID(streams []AstraStream) string {
	used := make(map[string]bool)

	for _, stream := range streams {
		id := strings.TrimSpace(stream.ID)
		if id != "" {
			used[id] = true
		}
	}

	for i := 1; i <= 9999; i++ {
		id := fmt.Sprintf("s%03d", i)
		if !used[id] {
			return id
		}
	}

	return "s9999"
}

func StreamServiceName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	if ContainsCyrillic(name) {
		return TranslitCyrillic(name)
	}

	return name
}

func ContainsCyrillic(value string) bool {
	for _, r := range value {
		if unicode.In(r, unicode.Cyrillic) {
			return true
		}
	}

	return false
}

func TranslitCyrillic(value string) string {
	replacer := strings.NewReplacer(
		"А", "A",
		"Б", "B",
		"В", "V",
		"Г", "G",
		"Д", "D",
		"Е", "E",
		"Ё", "Yo",
		"Ж", "Zh",
		"З", "Z",
		"И", "I",
		"Й", "Y",
		"К", "K",
		"Л", "L",
		"М", "M",
		"Н", "N",
		"О", "O",
		"П", "P",
		"Р", "R",
		"С", "S",
		"Т", "T",
		"У", "U",
		"Ф", "F",
		"Х", "Kh",
		"Ц", "Ts",
		"Ч", "Ch",
		"Ш", "Sh",
		"Щ", "Sch",
		"Ъ", "",
		"Ы", "Y",
		"Ь", "",
		"Э", "E",
		"Ю", "Yu",
		"Я", "Ya",

		"а", "a",
		"б", "b",
		"в", "v",
		"г", "g",
		"д", "d",
		"е", "e",
		"ё", "yo",
		"ж", "zh",
		"з", "z",
		"и", "i",
		"й", "y",
		"к", "k",
		"л", "l",
		"м", "m",
		"н", "n",
		"о", "o",
		"п", "p",
		"р", "r",
		"с", "s",
		"т", "t",
		"у", "u",
		"ф", "f",
		"х", "kh",
		"ц", "ts",
		"ч", "ch",
		"ш", "sh",
		"щ", "sch",
		"ъ", "",
		"ы", "y",
		"ь", "",
		"э", "e",
		"ю", "yu",
		"я", "ya",
	)

	return replacer.Replace(value)
}

//  ██████╗  █████╗ ██████╗ ██████╗  █████╗  ██████╗ ███████╗
// ██╔════╝ ██╔══██╗██╔══██╗██╔══██╗██╔══██╗██╔════╝ ██╔════╝
// ██║  ███╗███████║██████╔╝██████╔╝███████║██║  ███╗█████╗
// ██║   ██║██╔══██║██╔══██╗██╔══██╗██╔══██║██║   ██║██╔══╝
// ╚██████╔╝██║  ██║██║  ██║██████╔╝██║  ██║╚██████╔╝███████╗
//  ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═════╝ ╚═╝  ╚═╝ ╚═════╝ ╚══════╝

func IsValidLicense(value string) bool {
	if len([]rune(value)) != 32 {
		return false
	}

	for _, r := range value {
		if r >= '0' && r <= '9' {
			continue
		}

		if r >= 'a' && r <= 'f' {
			continue
		}

		if r >= 'A' && r <= 'F' {
			continue
		}

		return false
	}

	return true
}

func FormatAdapterCounter(value int) string {
	if value > 100 {
		return "99+"
	}

	return fmt.Sprintf("%d", value)
}

func NormalizeAdapterSignal(value int) int {
	if value <= 100 {
		return value
	}

	if value <= 0 {
		return 0
	}

	return 65535 / value
}
