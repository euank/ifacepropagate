package case02

type If1 interface {
	Method1()
	Method2()
}

type If2 interface {
	Method3()
	Method4()
}

type partialOverride struct {
	If1
}

// Embed If1, but only override Method1 on it, override Method3 if the inner one implements If2

func (p *partialOverride) Method1() {}
func (p *partialOverride) Method3() {}

func new(if1 If1) If1 {
	return &partialOverride{if1}
}
