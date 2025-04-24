package main

import (
	"log"
	"reflect"

	"github.com/Lexographics/go-postmangen"
)

type CreateUserRequest struct {
	Username string `json:"username" description:"The desired username" example:"johndoe"`
	Email    string `json:"email" description:"User's email address" example:"john.doe@example.com"`
	IsAdmin  bool   `json:"is_admin" example:"false"`
	Age      int    `json:"age"`
}

type GetUserRequest struct {
	UserID   string `param:"userId" description:"ID of the user to retrieve" example:"user-abc-123"`
	Format   string `query:"format" description:"Optional response format (e.g., 'short')" example:"full"`
	TenantID string `query:"tenant_id"`
}

func main() {
	pg := postmangen.NewPostmanGen("My API Example Collection", "Generated collection from examples/main.go")

	// Add Collection Variables (Optional)
	pg.AddVariable("base_url", "http://localhost:8080/api/v1")
	pg.AddVariable("token", "YOUR_INITIAL_JWT_TOKEN")

	// Add Default Placeholders (Optional)
	pg.AddPlaceholder("tenant_id", "default-tenant-001")

	log.Println("Registering routes...")

	// Register User Creation (POST /users)
	err := pg.Register(map[string]any{
		"method":    "POST",
		"path":      "/users",
		"inputType": reflect.TypeOf(CreateUserRequest{}),
	})
	if err != nil {
		log.Fatalf("Failed to register POST /users: %v", err)
	}
	log.Println("Registered POST /users")

	// Register Get User (GET /users/:userId)
	err = pg.Register(map[string]any{
		"method":    "GET",
		"path":      "/users/:userId",
		"inputType": reflect.TypeOf(GetUserRequest{}),
	})
	if err != nil {
		log.Fatalf("Failed to register GET /users/:userId: %v", err)
	}
	log.Println("Registered GET /users/:userId")

	// Generating the Collection
	outputFilename := "./example_collection.json"
	log.Printf("Generating Postman collection to %s...\n", outputFilename)

	err = pg.WriteToFile(outputFilename)
	if err != nil {
		log.Fatalf("Failed to write Postman collection: %v", err)
	}

	log.Printf("Postman collection generated successfully to %s!\n", outputFilename)
}
