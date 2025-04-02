package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type Todo struct {
	ID        int    `json:"_id,omitempty"`
	Completed bool   `json:"completed"`
	Body      string `json:"body"`
}

var db *sql.DB

func main() {
	fmt.Println("Starting Todo App!")

	err := godotenv.Load(".env")

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	initDB()
	defer db.Close()

	createTable()

	app := fiber.New()

	// app.Use(cors.New(cors.Config{
	// 	AllowOrigins: "http://localhost:5173",
	// 	AllowHeaders: "Origin,Content-Type,Accept",
	// }))

	PORT := os.Getenv("PORT")

	if PORT == "" {
		PORT = "5000"
	}

	if os.Getenv("ENV") == "production" {
		app.Static("/", "./client/dist")
	}

	app.Get("/api/todos", func(c *fiber.Ctx) error {
		todos, err := getAllTodos()

		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch todos"})
		}
		return c.Status(200).JSON(todos)
	})

	app.Post("/api/todos", func(c *fiber.Ctx) error {
		todo := &Todo{}

		if err := c.BodyParser(todo); err != nil {
			return err
		}

		if todo.Body == "" {
			return c.Status(400).JSON(fiber.Map{"msg": "Todo body is required"})
		}

		createdTodo, err := createTodo(todo.Body)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to create todo"})
		}

		return c.Status(201).JSON(createdTodo)
	})

	app.Patch("/api/todos/:id", func(c *fiber.Ctx) error {

		id, err := strconv.Atoi(c.Params("id"))

		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid ID Format"})
		}

		updatedTodo, err := markTodoCompleted(id)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.Status(404).JSON(fiber.Map{"error": "Todo not found!"})
			}

			return c.Status(500).JSON(fiber.Map{"error": "Failed to update todo"})
		}

		return c.Status(200).JSON(updatedTodo)
	})

	app.Delete("/api/todos/:id", func(c *fiber.Ctx) error {

		id, err := strconv.Atoi(c.Params("id"))

		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid ID Format"})
		}

		err = deleteTodo(id)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.Status(404).JSON(fiber.Map{"error": "Todo not found"})
			}
			return c.Status(500).JSON(fiber.Map{"error": "Failed to delete todo"})
		}

		return c.Status(200).JSON(fiber.Map{"success": true})
	})

	log.Fatal(app.Listen(":" + PORT))
}

func initDB() {
	var err error
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Successfully connected to PostgreSQL!")
}

func createTable() {
	query := `
		CREATE TABLE IF NOT EXISTS todos (
			id SERIAL PRIMARY KEY,
			body TEXT NOT NULL,
			completed BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}

func getAllTodos() ([]Todo, error) {
	query := "SELECT id, body, completed FROM todos ORDER BY created_at DESC"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var todo Todo
		err := rows.Scan(&todo.ID, &todo.Body, &todo.Completed)
		if err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}

	return todos, nil
}

func createTodo(body string) (*Todo, error) {
	query := "INSERT INTO todos (body) VALUES ($1) RETURNING id, body, completed"
	var todo Todo
	err := db.QueryRow(query, body).Scan(&todo.ID, &todo.Body, &todo.Completed)
	if err != nil {
		return nil, err
	}
	return &todo, nil
}

func markTodoCompleted(id int) (*Todo, error) {
	query := "UPDATE todos SET completed = TRUE WHERE id = $1 RETURNING id, body, completed"
	var todo Todo
	err := db.QueryRow(query, id).Scan(&todo.ID, &todo.Body, &todo.Completed)
	if err != nil {
		return nil, err
	}
	return &todo, nil
}

func deleteTodo(id int) error {
	query := "DELETE FROM todos WHERE id = $1"
	result, err := db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
