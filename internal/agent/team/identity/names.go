// Purpose: Assign unique readable team names from a catalog with numbered extensions.
// Pattern: Pure catalog with creation-time random selection.
// Usage: SQLite registration chooses an unused name; identities and prompts reuse it.
// Related: internal/agent/team/identity/member.go, internal/storage/sqlite/team_registration.go.
package identity

import (
	"crypto/rand"

	"math/big"
	"strconv"
)

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
	if index < 0 {
		return ""
	}
	name := names[index%len(names)]
	if round := index/len(names) + 1; round > 1 {
		return name + "-" + strconv.Itoa(round)
	}
	return name
}

func Choose(used []string) (string, error) {
	taken := make(map[string]struct{}, len(used))
	for _, name := range used {
		taken[name] = struct{}{}
	}
	var available [len(names)]uint8
	count := 0
	for i, name := range names {
		if _, exists := taken[name]; !exists {
			available[count] = uint8(i)
			count++
		}
	}
	if count == 0 {
		for i := len(names); ; i++ {
			name := Name(i)
			if _, exists := taken[name]; !exists {
				return name, nil
			}
		}
	}
	index, err := rand.Int(rand.Reader, big.NewInt(int64(count)))
	if err != nil {
		return "", err
	}
	return Name(int(available[index.Int64()])), nil
}
