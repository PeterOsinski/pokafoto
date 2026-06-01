//go:build ignore
package main
import (
    "database/sql"
    "fmt"
    "os"
    _ "modernc.org/sqlite"
)
func main() {
    os.MkdirAll("/tmp/kilo", 0755)
    os.Remove("/tmp/kilo/vacuum-test.db")
    defer os.Remove("/tmp/vacuum-test-source.db")
    db, _ := sql.Open("sqlite", "/tmp/vacuum-test-source.db?_pragma=journal_mode(WAL)")
    defer db.Close()
    db.Exec("CREATE TABLE IF NOT EXISTS test (id INTEGER)")
    db.Exec("INSERT INTO test VALUES (1)")
    dest := "/tmp/kilo/vacuum-test.db"
    if _, err := db.Exec(fmt.Sprintf("VACUUM INTO '%s'", dest)); err != nil {
        fmt.Println("QUOTED FAIL:", err)
    } else {
        info, _ := os.Stat(dest)
        fmt.Println("QUOTED OK, size:", info.Size())
        os.Remove(dest)
    }
    if _, err := db.Exec(fmt.Sprintf("VACUUM INTO %s", dest)); err != nil {
        fmt.Println("UNQUOTED FAIL:", err)
    } else {
        info, _ := os.Stat(dest)
        fmt.Println("UNQUOTED OK, size:", info.Size())
        os.Remove(dest)
    }
}
