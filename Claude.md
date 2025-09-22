# Claude's Go Backend Development Guide

## 1. Core Philosophy

This document provides guidance for developing the Go backend of the AI-Powered Code Knowledge Platform. The primary goal is to build a simple, maintainable, and performant MVP.

-   **Simplicity Over Complexity:** We favor the Go standard library, especially `net/http`, over heavy frameworks. Every dependency must justify its existence.
-   **Explicit is Better than Implicit:** Dependencies should be passed explicitly. Avoid global state and magic.
-   **Modularity:** Keep components decoupled through clear interfaces. This is crucial for testability and future flexibility (e.g., swapping out AI models or databases).
-   **Focus on the MVP:** Prioritize the core features: code ingestion, AI-powered documentation generation, and a Q&A API. [cite_start]Features like complex knowledge graph visualizations or project management integrations are for later iterations[cite: 42, 49].

## 2. Project Structure

[cite_start]Organize the codebase to maintain a clear separation of concerns[cite: 250].

/
├── cmd/
│   ├── server/main.go      # Main application for the web server
│   └── cli/main.go         # Main application for the CLI tool
├── internal/
│   ├── analysis/           # Code parsing, chunking, and knowledge extraction
│   ├── api/                # HTTP handlers, routing, and server setup
│   ├── ai/                 # AI model integration (OpenAI client, interfaces)
│   ├── qa/                 # Q&A logic, embeddings, and RAG implementation
│   └── store/              # Data storage interfaces and implementations (in-memory, DB)
├── web/
│   ├── templates/          # HTML templates for the frontend
│   └── static/             # CSS, JS (if any)
├── go.mod
└── go.sum


## 3. HTTP Service Design

Our approach is inspired by the "How I Write HTTP Services in Go" philosophy.

-   **Server Struct:** Create a central `Server` struct in the `api` package. This struct will hold all application dependencies like the database connection, logger, and AI client.

    ```go
    // internal/api/server.go
    type Server struct {
        router   *http.ServeMux
        aiClient ai.Service
        store    store.Store
        // add other dependencies like a logger
    }

    func NewServer(aiClient ai.Service, store store.Store) *Server {
        s := &Server{
            router:   http.NewServeMux(),
            aiClient: aiClient,
            store:    store,
        }
        s.routes()
        return s
    }
    ```

-   **Handlers as Methods:** HTTP handlers should be methods on the `Server` struct. This provides them with typed, compile-time checked access to the application's dependencies.

    ```go
    // internal/api/handlers.go
    func (s *Server) handleQuery() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            // Access dependencies via s.aiClient, s.store etc.
            // Use helper functions for writing JSON responses or rendering templates.
        }
    }
    ```

-   **Routing:** Use the standard library `http.ServeMux` for routing in the `routes()` method. Avoid third-party routers unless absolutely necessary for features like path parameters.

    ```go
    // internal/api/routes.go
    func (s *Server) routes() {
        s.router.HandleFunc("POST /api/query", s.handleQuery())
        s.router.HandleFunc("GET /doc/{id}", s.handleGetDocumentation())
    }
    ```

## 4. AI & Q&A Integration

-   [cite_start]**AI Service Interface:** Define an interface for all interactions with LLMs in `internal/ai/ai.go`[cite: 89]. [cite_start]The initial implementation will use the OpenAI API, but this interface allows for future expansion to other models (local or cloud-based)[cite: 89, 91].

    ```go
    // internal/ai/ai.go
    type Service interface {
        GenerateDocumentation(codeSnippet string) (string, error)
        AnswerQuestion(question string, context []string) (string, error)
        GenerateEmbeddings(chunks []string) ([][]float32, error)
    }
    ```

-   [cite_start]**Retrieval-Augmented Generation (RAG):** The Q&A logic resides in `internal/qa`[cite: 255].
    1.  [cite_start]**Chunking:** Code from `internal/analysis` will be chunked into logical pieces (e.g., by function)[cite: 160].
    2.  [cite_start]**Embedding:** Use the `ai.Service` to generate embeddings for each chunk[cite: 160].
    3.  [cite_start]**Storage & Retrieval:** For the MVP, store vectors in a simple in-memory slice or map within the `store` package[cite: 99, 102]. [cite_start]Implement a brute-force cosine similarity search for retrieval[cite: 102, 166]. This is sufficient for the initial scope.
    4.  [cite_start]**Prompting:** Construct a clear prompt for the LLM that includes the user's question and the retrieved code chunks as context[cite: 161, 170, 171].

## 5. CLI Tool

-   [cite_start]Use the `cobra` library to build the CLI in `cmd/cli/`[cite: 207, 251].
-   [cite_start]For the MVP, the CLI acts as a thin client[cite: 205]. [cite_start]It makes HTTP requests to the running backend service[cite: 204]. Do not duplicate business logic from the `internal` packages in the CLI.

## 6. Testing

-   [cite_start]Write unit tests for core logic in `analysis`, `qa`, and `ai`[cite: 125].
-   Use the `net/http/httptest` package to write tests for your HTTP handlers in the `api` package. This allows you to test handlers without spinning up a real server.