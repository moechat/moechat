package main
import (
	"io"
	"crypto/sha1"
	"fmt"
	"math/rand"
)

func generateToken() (string) {
	return hashString(randomString(20))
}

/* Hashes and returns an input string using SHA1 */
func  hashString(input string) (string) {
	h := sha1.New()
	io.WriteString(h, input)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func randomString (l int) string {
    bytes := make([]byte, l)
    for i:=0 ; i<l ; i++ {
        bytes[i] = byte(randInt(65,90))
    }
    return string(bytes)
}

func randInt(min int , max int) int {
    return min + rand.Intn(max-min)
}

/*
func main() {
	fmt.Println("hi: " + hashString("hi"))
    rand.Seed( time.Now().UTC().UnixNano())

	for i:=0; i<10; i++ {
		fmt.Println(generateToken());
	}
}
*/
