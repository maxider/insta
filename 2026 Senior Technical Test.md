# BoWatt Technical Test

## Objective

Create a small, Instagram-like web application to demonstrate your technical skills in full-stack web development. The site should let users upload images anonymously and view newly uploaded images live in a feed.

## Requirements

### Frontend (React / TypeScript)

Create a simple, clean, and responsive React/TypeScript-based frontend. The frontend should let users:

- Upload new images with a title and tags.
- See uploaded images with their titles and tags. This feed should be updated live via a websocket.
- Filter the feed by tags.

### Backend (Go/Another compiled statically typed language)

Develop an API to handle requests from the frontend. You may use any libraries you see fit. Errors should be handled/reported appropriately. The API should:

- Serve an initial set of images through a REST endpoint.
- Accept uploads over a `/uploads` endpoint (storing files locally is fine).
- Provide and manage websocket connections for notifying users of newly uploaded images.

## Submission Requirements

**AI:** Use of AI/LLM code generation when solving this technical test is forbidden. We need to see your own abilities to solve issues and find solutions. All code submitted is your responsibility.

**Codebase:** Provide a link to any repos created for your work.

**Documentation:**

1. Instructions to set up and run the project.
2. Provide a brief explanation of the architecture and design decisions. Include any plans you had that you may not have had time for.
3. Simple API documentation.

Happy coding!

## Bonus Points

If you have time and feel up to an extra challenge, pick and choose any of the following items to include in your submission:

- Use Docker to containerize the application
- Implement a basic caching mechanism to speed up repeated queries
- Add unit tests for key components
- Queuing for file uploads
- Infinite-scroll feed
- Normalise uploaded images
- Polished UI
