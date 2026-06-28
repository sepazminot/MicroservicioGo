package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Definir estructuras FUERA de main()
type User struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Conexión a la Base de Datos
	dbURL := os.Getenv("DATABASE_URL")
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Pool equilibrado exactamente a 20 conexiones (Igual que Express y Phoenix)
	db.SetMaxOpenConns(20)                  // Límite máximo de conexiones activas (Igual a Express y Phoenix)
	db.SetMaxIdleConns(20)                  // Mantenerlas abiertas en reposo para evitar crear sockets constantemente
	db.SetConnMaxIdleTime(5 * time.Minute)  // Sincronizado exactamente con idleTimeoutMillis de Express (300000 ms)
	db.SetConnMaxLifetime(30 * time.Minute) // Un tiempo de vida largo (ej. 30 min) para mitigar la rotación destructiva bajo estrés

	// Verificar la salud de la conexión antes de levantar el servidor
	if err := db.Ping(); err != nil {
		panic("No se pudo conectar a la base de datos: " + err.Error())
	}

	// ==================== USUARIOS ====================
	// GET - Obtener usuario
	r.GET("/users/:id", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		var user User

		err := db.QueryRow("SELECT id, email, password FROM users WHERE id = $1", id).
			Scan(&user.ID, &user.Email, &user.Password)

		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
			return
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener usuario"})
			return
		}

		c.JSON(http.StatusOK, user)
	})

	// POST - Crear usuario
	r.POST("/users", func(c *gin.Context) {
		var req User
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
			return
		}

		var newUser User
		err := db.QueryRow("INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id, email, password", req.Email, req.Password).
			Scan(&newUser.ID, &newUser.Email, &newUser.Password)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear usuario"})
			return
		}

		c.JSON(http.StatusCreated, newUser)
	})

	// PUT - Actualizar usuario (Payload completo directo, igual que Express)
	r.PUT("/users/:id", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		var req User
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
			return
		}

		var updatedUser User
		err := db.QueryRow("UPDATE users SET email = $1, password = $2 WHERE id = $3 RETURNING id, email, password", req.Email, req.Password, id).
			Scan(&updatedUser.ID, &updatedUser.Email, &updatedUser.Password)

		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
			return
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar usuario"})
			return
		}

		c.JSON(http.StatusOK, updatedUser)
	})

	// DELETE - Eliminar usuario
	r.DELETE("/users/:id", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))

		result, err := db.Exec("DELETE FROM users WHERE id = $1", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar usuario"})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Usuario eliminado"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Servidor corriendo en puerto %s", port)
	r.Run(":" + port)
}
