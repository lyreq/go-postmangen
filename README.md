# Go PostmanGen

`go-postmangen` is a Go library designed to generate Postman collections programmatically from your Go code. It uses Go struct definitions and tags to automatically create Postman requests, including URLs, methods, headers, path variables, query parameters, and request bodies (JSON or form-data).

This simplifies the process of keeping your Postman collections synchronized with your Go API implementation.

## Features

*   **Automatic Request Generation:** Define your API request structures as Go structs and let `go-postmangen` generate the corresponding Postman requests.
*   **Struct Tag Mapping:** Use familiar struct tags (`json`, `form`, `formFile`, `query`, `param`) to map struct fields to different parts of an HTTP request.
*   **JSON & Form-Data Support:** Automatically generates requests with `application/json` or `multipart/form-data` content types based on struct tags.
*   **Path & Query Parameters:** Handles URL path parameters (e.g., `/users/:id`) and query parameters.
*   **Placeholder Values:** Use `example` tags or define default placeholders for request fields.
*   **Collection Variables:** Easily add Postman collection variables (like `base_url`).
*   **Folder Organization:** Automatically organizes requests into folders based on URL paths.
*   **Default Authentication:** Includes Bearer Token authentication (`{{token}}`) by default.
*   **Simple API:** Easy-to-use API for creating, configuring, and generating collections.

## Installation

```bash
go get github.com/Lexographics/go-postmangen
```

## Core Concepts

*   **`PostmanGen`:** The main struct that manages the creation of the Postman collection. You initialize it, configure it, register your API endpoints, and finally generate the collection file.
*   **Request Structs:** You define Go structs that represent the data structure for your API requests (path parameters, query parameters, request body).
*   **Struct Tags:** Special tags added to the fields of your request structs tell `go-postmangen` how to map each field to the generated Postman request. Supported tags are:
    *   `json:"<key_name>"`: Maps the field to a key in a JSON request body.
    *   `form:"<key_name>"`: Maps the field to a key in a `multipart/form-data` request (text field).
    *   `formFile:"<key_name>"`: Maps the field to a key in a `multipart/form-data` request (file field).
    *   `query:"<key_name>"`: Maps the field to a URL query parameter.
    *   `param:"<key_name>"`: Maps the field to a URL path parameter (the `<key_name>` should match the parameter name in the path, e.g., `:id`).
    *   `description:"<text>"`: Adds a description to the parameter/field in Postman.
    *   `example:"<value>"`: Provides an example value to be used as a placeholder in the generated request.
*   **Placeholders:** `go-postmangen` populates the generated requests with placeholder values. The order of precedence is:
    1.  Value from the `example:"..."` tag.
    2.  Value from a default placeholder set via `AddPlaceholder()`.
    3.  The Go zero value for the field's type (e.g., `0` for `int`, `""` for `string`, `false` for `bool`).

## User Manual & Usage

### 1. Initialization

Create a new `PostmanGen` instance. This initializes a Postman collection with a default `{{base_url}}` variable and Bearer Token authentication using `{{token}}`.

```go
package main

import (
	"log"
	"reflect"

	"github.com/Lexographics/go-postmangen"
)

func main() {
	pg := postmangen.NewPostmanGen("My API Collection", "Generated collection for My API")

	// Add a value for the default base_url variable
	pg.AddVariable("base_url", "http://localhost:8080")

	// (Register routes here - see next steps)

	// Generate the collection
	err := pg.WriteToFile("my_api.postman_collection.json")
	if err != nil {
		log.Fatalf("Failed to write Postman collection: %v", err)
	}
	log.Println("Postman collection generated successfully!")
}

```

### 2. Adding Collection Variables (Optional)

You can add more collection-level variables besides the default `base_url` and `token`.

```go
pg.AddVariable("api_key", "YOUR_DEFAULT_API_KEY")
```

### 3. Adding Default Placeholders (Optional)

Define default values for specific field names used across different request structs. This is useful if you have common fields like `user_id` or `tenant_id`.

```go
pg.AddPlaceholder("user_id", "default-user-123")
pg.AddPlaceholder("page", "1")
pg.AddPlaceholder("limit", "20")
```
If a field named `user_id` doesn't have an `example` tag, `go-postmangen` will use `"default-user-123"` as its placeholder value.

### 4. Defining Request Structs

Define Go structs representing your API endpoints' inputs. Use struct tags to specify how each field maps to the Postman request.

**Example: User Creation (JSON Body)**

```go
type CreateUserRequest struct {
	Username string `json:"username" description:"The desired username" example:"johndoe"`
	Email    string `json:"email" description:"User's email address" example:"john.doe@example.com"`
	IsAdmin  bool   `json:"is_admin" example:"false"`
	Age      int    `json:"age"` // No example, will use Go zero value (0)
}
```

**Example: Get User (Path Parameter & Query Parameter)**

```go
type GetUserRequest struct {
	UserID   string `param:"userId" description:"ID of the user to retrieve" example:"user-abc-123"`
	Format   string `query:"format" description:"Optional response format (e.g., 'short')" example:"full"`
	TenantID string `query:"tenant_id"` // No example, might use AddPlaceholder or zero value ""
}

```

**Example: Upload Profile Picture (Form-Data)**

```go
type UploadPictureRequest struct {
	UserID      string `param:"userId" description:"ID of the user" example:"user-abc-123"`
	Description string `form:"description" description:"Optional description for the image" example:"My profile picture"`
	ProfilePic  string `formFile:"profile_pic" description:"The profile picture file to upload"`
}
```

### 5. Registering Routes

Register each endpoint by providing its HTTP method, path, and the `reflect.Type` of its corresponding request struct.

The `Register` method expects a `map[string]any` for the spec currently.

```go
	// Register User Creation (POST /users)
	err := pg.Register(map[string]any{
		"method":    "POST",
		"path":      "/users",
		"inputType": reflect.TypeOf(CreateUserRequest{}),
	})
	if err != nil {
		log.Fatalf("Failed to register POST /users: %v", err)
	}

	// Register Get User (GET /users/:userId)
	err = pg.Register(map[string]any{
		"method":    "GET",
		"path":      "/users/:userId",
		"inputType": reflect.TypeOf(GetUserRequest{}),
	})
	if err != nil {
		log.Fatalf("Failed to register GET /users/:userId: %v", err)
	}

	// Register Upload Picture (POST /users/:userId/picture)
	err = pg.Register(map[string]any{
		"method":    "POST",
		"path":      "/users/:userId/picture",
		"inputType": reflect.TypeOf(UploadPictureRequest{}),
	})
	if err != nil {
		log.Fatalf("Failed to register POST /users/:userId/picture: %v", err)
	}
```

*   **Path Parameters:** Use the `:paramName` syntax in the `path` string (e.g., `/users/:userId`). Ensure you have a corresponding field in your struct tagged with `param:"paramName"`.
*   **Request Body Type:** `go-postmangen` automatically determines the request body type:
    *   If any field has a `form:"..."` or `formFile:"..."` tag, it uses `multipart/form-data`.
    *   Otherwise, if any field has a `json:"..."` tag, it uses `application/json`.
    *   If neither `form`/`formFile` nor `json` tags are present but other tags (`query`, `param`) are, no request body is generated.
*   **Folder Structure:** Requests are automatically placed in folders based on the path structure. For example, `GET /users/:userId` and `POST /users/:userId/picture` will both be placed inside a `users` folder, with the latter inside a `:userId` subfolder(subfolder generation for url parameters might change later).

### 6. Generating the Collection

Finally, write the generated collection to a file.

```go
err := pg.WriteToFile("my_api.postman_collection.json")
if err != nil {
	log.Fatalf("Failed to write Postman collection: %v", err)
}
log.Println("Postman collection generated successfully!")
```

You can also write the collection to any `io.Writer`:

```go
// import "os"
// err := pg.Write(os.Stdout)
```

## Example

A runnable example showcasing the basic usage can be found in [`examples/main.go`](./examples/main.go).

## How it Works Internally

The library uses Go's reflection capabilities (`reflect` package) to inspect the fields and tags of the provided struct types. It maps these tags to the corresponding parts of a Postman request. It constructs the request URL, headers, body, parameters, and organizes them into items and folders within the Postman collection structure before serializing it to JSON.

## Contributing

Contributions are welcome! Please feel free to submit pull requests or open issues.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.