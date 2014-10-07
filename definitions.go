package main

/* -------------------------------------------- */
type Entity struct {
	Animations struct {
		A int
		B int
	}
	Angles struct {
		A struct {
			X float32
			Y float32
			Z float32
		}
		B struct {
			X float32
			Y float32
			Z float32
		}
	}
	Client   int
	Entities struct {
		A int
		B int
		C int
	}
	Entity struct {
		A int
		B int
	}
	Events struct {
		A int
		B int
	}
	Misc struct {
		A int
		B int
		C int
		D int
		E int
	}
	Model struct {
		A int
		B int
	}
	Origins struct {
		A struct {
			X float32
			Y float32
			Z float32
		}
		B struct {
			X float32
			Y float32
			Z float32
		}
	}
	Powerups int
	Time     struct {
		A int
		B int
	}
	Trajectories struct {
		A Trajectory
		B Trajectory
	}
	Weapon int
}

type Trajectory struct {
	Base struct {
		X float32
		Y float32
		Z float32
	}
	Delta struct {
		X float32
		Y float32
		Z float32
	}
	Duration int
	Gravity  int
	Mode     int
	Time     int
}

/* -------------------------------------------- */
type Player struct {
	Ammunition struct {
		A int
		B int
		C int
		D int
		E int
		F int
		G int
		H int
		I int
		J int
		K int
		L int
		M int
		N int
		O int
		P int
	}
	Animations struct {
		A struct {
			A int
			B int
		}
		B struct {
			A int
			B int
		}
	}
	Attributes struct {
		A int
		B int
		C int
		D int
		E int
		F int
		G int
		H int
		I int
		J int
		K int
		L int
		M int
		N int
		O int
		P int
	}
	Client int
	Damage struct {
		A int
		B int
		C int
		D int
	}
	Delta struct {
		A int
		B int
		C int
	}
	Entities struct {
		A int
		B int
	}
	Entity struct {
		A int
	}
	Event struct {
		A int
		B int
		C int
	}
	Events struct {
		A int
		B int
	}
	External struct {
		A int
		B int
	}
	Grapple struct {
		X float32
		Y float32
		Z float32
	}
	Origin struct {
		X float32
		Y float32
		Z float32
	}
	Velocity struct {
		X float32
		Y float32
		Z float32
	}
	View struct {
		X float32
		Y float32
		Z float32
	}
	Misc struct {
		A int
		B int
		C int
		D int
		E int
		F int
	}
	Movement struct {
		A int
		B int
		C int
		D int
	}
	Powerups struct {
		A int
		B int
		C int
		D int
		E int
		F int
		G int
		H int
		I int
		J int
		K int
		L int
		M int
		N int
		O int
		P int
	}
	Time   int
	Vitals struct {
		A int
		B int
		C int
		D int
		E int
		F int
		G int
		H int
		I int
		J int
		K int
		L int
		M int
		N int
		O int
		P int
	}
	Weapon struct {
		A int
		B int
		C int
	}
}

/* -------------------------------------------- */
type Gamestate struct {
	Id       int
	Client   int
	Checksum int
}

type Snapshot struct {
	Time  int
	Delta int
	Flags int
	Blob  []byte
}

type Command struct {
	Id  int
	Str string
}
