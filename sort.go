package contacts

type ByName []*Contact

func (b ByName) Len() int           { return len(b) }
func (b ByName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ByName) Less(i, j int) bool { return compareDisplay(b[i], b[j]) }

type ByLastName []*Contact

func (b ByLastName) Len() int           { return len(b) }
func (b ByLastName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ByLastName) Less(i, j int) bool { return compareName(b[i], b[j]) }

type ByBirthday []*Contact

func (b ByBirthday) Len() int           { return len(b) }
func (b ByBirthday) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ByBirthday) Less(i, j int) bool { return compareBirthday(b[i], b[j]) }

func compareBirthday(lhs, rhs *Contact) bool {
	lb, rb := lhs.birthdayOrZero(), rhs.birthdayOrZero()
	if lb.Month() == rb.Month() {
		if lb.Day() == rb.Day() {
			if lb.Year() == rb.Year() {
				return compareDisplay(lhs, rhs)
			}
			return lb.Year() < rb.Year()
		}
		return lb.Day() < rb.Day()
	}
	return lb.Month() < rb.Month()
}

func compareDisplay(lhs, rhs *Contact) bool {
	if lhs == rhs {
		return false
	}
	if lhs == nil {
		return true
	}
	if rhs == nil {
		return false
	}
	return lhs.DisplayName() < rhs.DisplayName()
}

func compareName(lhs, rhs *Contact) bool {
	if lhs == rhs {
		return false
	}
	if lhs == nil {
		return true
	}
	if rhs == nil {
		return false
	}
	if lhs.Last == rhs.Last {
		if lhs.First == rhs.First {
			return compareDisplay(lhs, rhs)
		}
		return lhs.First < rhs.First
	}
	return lhs.Last < rhs.Last
}
