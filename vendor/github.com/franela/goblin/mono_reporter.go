package goblin

import ()

type Monochrome struct {
}

func (self *Monochrome) Red(text string) string {
	return "!" + text
}

func (self *Monochrome) Gray(text string) string {
	return text
}

func (self *Monochrome) Cyan(text string) string {
	return text
}

func (self *Monochrome) WithCheck(text string) string {
	return ">>>" + text
}

func (self *Monochrome) Green(text string) string {
	return text
}
