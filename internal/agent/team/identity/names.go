// Purpose: Assign unique readable team names from one immutable cross-cultural catalog.
// Pattern: Pure catalog with creation-time random selection.
// Usage: SQLite registration chooses an unused name; identities and prompts reuse it.
// Related: internal/agent/team/identity/member.go, internal/storage/sqlite/team_registration.go.
package identity

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"slices"
)

// Arrays cannot be const in Go; this private catalog is never mutated.
var names = [...]string{
	"Hana",
	"Ren",
	"Arjun",
	"Jisoo",
	"Minho",
	"Seojun",
	"Yuna",
	"Jiho",
	"Seoyeon",
	"Doyun",
	"Sumin",
	"Jiwon",
	"Hyunwoo",
	"Nari",
	"Taeyang",
	"Bora",
	"Junho",
	"Eunji",
	"Siwoo",
	"Haeun",
	"Kenji",
	"Yui",
	"Sora",
	"Haruto",
	"Aoi",
	"Yuto",
	"Sakura",
	"Takumi",
	"Mei",
	"Riku",
	"Akira",
	"Nanami",
	"Daiki",
	"Misaki",
	"Kaito",
	"Hina",
	"Naoki",
	"Ayaka",
	"Itsuki",
	"Minji",
	"Liwei",
	"Meilin",
	"Xinyi",
	"Haoran",
	"Yichen",
	"Zihan",
	"Jiahao",
	"Yuxin",
	"Zixuan",
	"Ruoxi",
	"Yutong",
	"Mingyu",
	"Wenhao",
	"Xiaoyu",
	"Zeyu",
	"Shihan",
	"Jingyi",
	"Chenyu",
	"Yiming",
	"Linyu",
	"Oliver",
	"Emma",
	"Noah",
	"Amelia",
	"Liam",
	"Olivia",
	"Ethan",
	"Sophia",
	"Lucas",
	"Charlotte",
	"Henry",
	"Grace",
	"Owen",
	"Alice",
	"Felix",
	"Clara",
	"Leo",
	"Ivy",
	"Oscar",
	"Ruby",
	"Anika",
	"Kiran",
	"Aarav",
	"Ishaan",
	"Vihaan",
	"Diya",
	"Priya",
	"Rohan",
	"Neha",
	"Aditya",
	"Kavya",
	"Rahul",
	"Asha",
	"Dev",
	"Nisha",
	"Varun",
	"Tara",
	"Vikram",
	"Leela",
	"Seungho",
}

func Name(index int) string {
	if index < 0 || index >= len(names) {
		return ""
	}
	return names[index]
}

func Choose(used []string) (string, error) {
	var available [len(names)]uint8
	count := 0
	for i, name := range names {
		if !slices.Contains(used, name) {
			available[count] = uint8(i)
			count++
		}
	}
	if count == 0 {
		return "", fmt.Errorf("no unused team member name is available")
	}
	index, err := rand.Int(rand.Reader, big.NewInt(int64(count)))
	if err != nil {
		return "", err
	}
	return Name(int(available[index.Int64()])), nil
}
