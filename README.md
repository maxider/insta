This project contains my results for the BoWatt technical test. A detailed writeup about my implementation can be found under FinalNotes.md where also things I'd like to add can be found.

# How to run

Run `docker compose up` from the root context and go to https://localhost. I'm using caddy with auto https so the browser will warn you of an insecure connection due to Caddy's own local certificate. I sadly did not have time to resolve this.

Rebuilding will wipe all data since volumes are not used.

## Backend Standalone

### Requirements
- go 1.27.3

The backend can be started by running `go run .` from the backend folder. The backend will use `localhost:8080` by default. To change the host run `go run . host:port`. The backend will create `db.db` and `./file-storage/` at its root.

## Frontend Standalone

  Install frontend with `npm install`
  
   The Frontend can be run with `npm run dev` from the insta folder(./frontend/insta). It can be accessed through "http://localhost:5173". It's hardcoded to use the backend on localhost:8080 through vite proxy routing.
