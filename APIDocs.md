Note on Caddy routing: when the backend runs through Caddy, `/uploads` and `/latest` are reached through `/api/...`.

- GET /images/{id} - fetches the image files from the backend through the id of the image returned from `/latest`. Returns the raw image bytes.
- POST /uploads - endpoint to upload files to. Uses multipart form data where the fields have to be named `title`, `tags`, and `file`. The tags are comma-separated. So for example `hello,world` would be the tags `hello` and `world`. The file has a maximum size of 10MB. Only image/jpeg, image/png, and image/webp are allowed. Returns `{"id": "<uuid>"}` with code 201
- GET /latest?limit=n - get the latest n images. The limit query param is not required, defaults to 20, and has a maximum value of 20. Zombie code for filtering also exists which would use the `?filter=tag1,tag2,tag3` syntax. Returns 200 with `{"metadata": [{ id, title, tags, filesize, mimetyp, size_x, size_y, created_at }]}`
- Errors: `{"statusCode": <code>, "msg": "<message>"}`
