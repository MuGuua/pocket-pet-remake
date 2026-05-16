package pet

import "errors"

var (
	ErrPetNotFound     = errors.New("pet not found")
	ErrInvalidLineup   = errors.New("invalid lineup")
	ErrDuplicateLineup = errors.New("duplicate lineup pet")
)

type Pet struct {
	PetUID   uint64
	PetID    uint32
	Level    uint32
	Exp      uint64
	Quality  uint32
	HP       uint32
	HPMax    uint32
	ATK      uint32
	DEF      uint32
	SPD      uint32
	SkillIDs []uint32
	InLineup bool
}

type LineupPet struct {
	PetUID uint64
	PetID  uint32
	Level  uint32
	HP     uint32
	HPMax  uint32
}
