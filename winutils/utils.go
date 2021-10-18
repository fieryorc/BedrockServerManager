package winutils

func NewLine() string {
	return "\r\n"
}

func AddNewLine(str string) string {
	return str + NewLine()
}
