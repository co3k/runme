package main

import (
	"log"
	"database/sql"
	"time"

	"golang.org/x/crypto/bcrypt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

var jwtKey = []byte("please_change_this_key")

var db *sql.DB

func init() {
  var err error
  db, err = sql.Open("sqlite3", "./users.db")
  if err != nil {
    panic(err)
  }
  
	statement, _ := db.Prepare("CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, username TEXT UNIQUE, password TEXT)")
	statement.Exec()
}

func main() {
	app := fiber.New()
	app.Use(logger.New())

	app.Post("/register", func (c *fiber.Ctx) error {
		var data map[string]string

		if err := c.BodyParser(&data); err != nil {
			c.Status(500)
			return c.JSON(fiber.Map{
				"message": "internal server error",
			})
		}

		username := data["username"]
		password := data["password"]

		if username == "" || password == "" {
			c.Status(400)
			return c.JSON(fiber.Map{
				"message": "username or password is empty",
			})
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 8)
		if err != nil {
			c.Status(500)
			return c.JSON(fiber.Map{
				"message": "internal server error",
			})
		}

		statement, err := db.Prepare("INSERT INTO users (username, password) VALUES (?, ?)")
		if err != nil {
			c.Status(500)
			return c.JSON(fiber.Map{
				"message": "internal server error",
			})
		}
		result, err := statement.Exec(&username, &hashedPassword)
		if err != nil {
			c.Status(500)
			return c.JSON(fiber.Map{
				"message": "internal server error",
			})
		}

		id, _ := result.LastInsertId()

		token := jwt.New(jwt.SigningMethodHS256)
		claims := token.Claims.(jwt.MapClaims)
		claims["identity"] = id
		claims["exp"] = time.Now().Add(time.Hour * 72).Unix()

		t, err := token.SignedString([]byte(jwtKey))
		if err != nil {
			c.Status(500)
			return c.JSON(fiber.Map{
				"message": "internal server error",
			})
		}

		return c.JSON(fiber.Map{
			"token": t,
		})
	})

	app.Post("/login", func (c *fiber.Ctx) error {
		var data map[string]string

		if err := c.BodyParser(&data); err != nil {
			c.Status(500)
			return c.JSON(fiber.Map{
				"message": "internal server error",
			})
		}

		username := data["username"]
		password := data["password"]

		if username == "" || password == "" {
			c.Status(400)
			return c.JSON(fiber.Map{
				"message": "username or password is empty",
			})
		}

		row := db.QueryRow("SELECT id, username, password FROM users WHERE username = ?", username)

		var user User
		err := row.Scan(&user.ID, &user.Username, &user.Password)
		if err != nil {
			c.Status(500)
			return c.JSON(fiber.Map{
				"message": "internal server error",
			})
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
			c.Status(400)
			return c.JSON(fiber.Map{
				"message": "incorrect password",
			})
		}

		token := jwt.New(jwt.SigningMethodHS256)
		claims := token.Claims.(jwt.MapClaims)
		claims["identity"] = user.ID
		claims["exp"] = time.Now().Add(time.Hour * 72).Unix()

		t, err := token.SignedString([]byte(jwtKey))
		if err != nil {
			c.Status(500)
			return c.JSON(fiber.Map{
				"message": "internal server error",
			})
		}

		return c.JSON(fiber.Map{
			"token": t,
		})
	})

	log.Fatal(app.Listen(":3000"))
}
