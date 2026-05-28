package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// Definir estructuras FUERA de main()
type User struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type PostUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func main() {
	gin.SetMode(gin.ReleaseMode)

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		// Fallback para desarrollo local
		connStr = "user=postgres password=1234 dbname=userdb sslmode=disable"
	}
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(20)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	// Verificar conexión
	if err = db.Ping(); err != nil {
		log.Fatal("Error al conectar a la base de datos:", err)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	// ---------------------- User (CORREGIDO)
	r.GET("/users/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
			return
		}

		var user User
		err = db.QueryRow("SELECT id, email, password FROM users WHERE id = $1", id).
			Scan(&user.ID, &user.Email, &user.Password)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		c.JSON(http.StatusOK, user)
	})

	r.POST("/users", func(c *gin.Context) {
		var req PostUserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos " + err.Error()})
			return
		}

		var id int64
		err = db.QueryRow("INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id",
			req.Email, req.Password).Scan(&id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear el usuario " + err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id, "email": req.Email, "password": req.Password})
	})

	r.DELETE("/users/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
			return
		}

		result, err := db.Exec("DELETE FROM users WHERE id = $1", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar el usuario " + err.Error()})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"id": id, "message": "Usuario eliminado"})
	})

	r.PUT("/users/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
			return
		}

		var req PostUserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos " + err.Error()})
			return
		}

		result, err := db.Exec("UPDATE users SET email = $2, password = $3 WHERE id = $1",
			id, req.Email, req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar el usuario " + err.Error()})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"id": id, "email": req.Email, "password": req.Password})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Servidor corriendo en puerto %s", port)
	r.Run(":" + port)
}
