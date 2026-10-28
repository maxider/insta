# General prelude
I had a lot of fun with this project. Sadly, I did not manage to complete all of the goals set. I hit my limits with the websocket and tag filtering.

The roadblock I hit with the filtering was the SQL query. I could possibly get a hacky filtering process going where I fetch all the images, then filter on that, but that's not ok for obvious reasons. I'm certain that all it takes is a solid query with joins, but without AI assistance, I can't manage that query. So, I'll leave it at this. Everything else should be in place so that the correct query will yield a satisfactory result.

I have a similar problem with the websocket, where I can't get the communication going without errors. I can communicate bidirectionally between backend and frontend, but not reliably across refreshes without the backend crashing. This is likely due to my naive implementation, assuming things about net/http and coder/websocket without really knowing them. I also assume that a solution to this is a more sophisticated websocket hub structure that handles the connections. I've committed my attempt at this on its own branch named `ws-attempt`.

Initially, I spent a lot of time on a sufficient error handling pattern, including logging. I'll elaborate on this in the architecture section. This was time well spent, as I benefited from it when running into errors. Sadly, I was not consistent with the level of detail in my errors, so some wrap errors in more detailed text while others just return the error. But through the architecture, I was able to never leak internal errors to the client.

# Architecture
For the backend I've decided to use net/http as the base and not use any packages for HTTP handling. This project only has a few endpoints and simple middleware, so it suffices. I'm serving the list of images to the frontend as an array of metadata, then the id is used to fetch the images through `<img src={id}>`. This hides the internal representation of the image, not leaking possible internal information. This also abstracts the URL from the underlying storage.

My backend also routes requests for images to its `ImageBlobStore` interface. With this I'm not dependent on one kind of file storage, but can easily replace it with others from the cloud like an Amazon S3 bucket or Azure Blob Storage. Testability is also enabled through that.

I'd love to have the same kind of abstraction for the database persistence, but did not manage to touch that within the given time. This would have allowed for a simple replacement of the database layer, only having to add a concretion of the interface.

For the logging I've decided to add logging middleware to every request. This has helped me immensely in debugging, helping me understand where things go wrong through the structured logging and the scattering of debug logs. I'm not fully satisfied with my implementation, as I've implemented this on `http.HandleFunc`. Instead I should have implemented it on `http.Handler` as this wraps all requests going through it instead of just annotated functions. I'd also like to add the status of the response to the logs, and possibly also the passed-in request body in debug mode. Logging the entire body might be bloaty and require some special cases, considering the file transfers. I'd also have to consider if it leaks confidential data.

As already alluded to in the introduction, I've added an `APIError` struct into my stack. Through this, with the addition of a custom `handleFunc`, I'm able to return errors from my handlers up to the middleware and decide there how to handle them. By wrapping information in this custom type I can also add a message to the client decoupled from the underlying system error. The system error can then contain any information relevant for debugging without it leaking to the end user. I also believe that this is more resilient error handling, as passing errors up the call chain is already something I'm used to. And one does not have to create custom writes to `http.ResponseWriter` on every error condition worthy of a notification. Sadly I was not enforcing the wrapping of user-facing errors consistently, as seen through some TODOs. I was more focused on getting the features in than creating clean error reporting on all occasions. But I hope that the parts I've applied it to demonstrate my capability here.

The uploading is done through multipart forms, which don't have to be ordered. I considered using plain JSON for this and passing the image along as a base64-encoded string, but I wanted to face the foreign territory and work with this. My implementation does not care about the order of the parts. I also believe that an expansion into multiple image uploads per request is not too difficult with this. I did not yet consider if that has notable benefits over just using multiple requests to upload multiple files. The tags are passed along as a comma-separated list. While the frontend does not have validation (something I would add with more time), the tag input I've used creates a new tag on comma usage. So from the input side this separation is fine.

The images are fetched through `/latest?limit=n`, which returns the latest n images. This endpoint also has initial capabilities for tag filtering (currently commented out) through the `?filter=a,b,c` query param. But as already explained in the introduction, I hit a roadblock with the filtered query. I'd also expand this endpoint with a cursor, which allows for an infinite scroll fetching past the first n images, as well as pagination. The cursor would be a simple additional query param `?cursor=id` which marks the last id served. While new uploads shift which images count as the latest n, a cursor against the id is resilient against this, as it directly points to an image. Pagination would work in a similar way: it could also be used for infinite scrolling, and it allows moving to older images on another page of the website.

## Data modeling

The data model I've decided to use splits an upload into two parts: the image metadata and the tags. These are stored separately in the DB. The many-to-many relationship between them gets enforced through a join table. In the image metadata I'm mainly storing the id, title, and filetype, but also some additional nice-to-have info, such as the filesize and the dimensions. The dimensions are not wired up but could be used for frontend rendering information. These additional columns are also nice info to have in general. If space is a concern I might consider dropping them. For the ids I deliberately chose UUIDv7 over serial ids. I'm not a fan of serial ids, as they leave gaps on deletions and do not distribute cleanly. UUIDs do not have these downsides, and v7 also allows for ordering due to the timestamp.

I thought about logic that catches duplicate images but decided against it at this level, as I'm too unsure whether duplicate uploads are ok here or not. My naive initial thought for this is a hash of the image. But this opens up a can of worms. How would one handle the deletion of a post where the image is referenced across many posts? Who has the final deletion power over the image?

The tags are stored very plainly, only with a name and a creation date. Through this, every tag only has to be stored once regardless of how often it is used. The reference between an image and its tags is done through the join table storing the image id and the tag name. If an image A has tags X, Y, and Z, there will be rows A-X, A-Y and A-Z. This makes the filter process a non-trivial join operation which also has to account for the correct application of the limit statement. As explained in the introduction, I did not manage to resolve this, so I left it out. All other things for the filtering should be in place.

## Frontend

The frontend uses [shadcn/ui](https://ui.shadcn.com/) as its component library. I'm using this as I don't want to spend my time designing and implementing components. Instead I want to get the logic going. But having nice visuals is also just nice.

The fetching of the backend data is done through [TanStack Query](https://tanstack.com/query/latest). I believe that an abstraction for the external state is essential: one has to consider data invalidation, updates, when to update and so on. Plain HTTP requests through fetch or axios, with local state management via `useState` or a state management library like [zustand](https://zustand.docs.pmnd.rs/), would grow fast in complexity and have a lot of edge cases that have to be handled. I also used [TanStack Form](https://tanstack.com/form/latest) for my upload and filter forms, as this attaches nicely to TanStack Query.

## Docker

The Docker setup is done in three parts: two dockerfiles for the backend and frontend, and the final Docker Compose file.

The backend dockerfile is straightforward: it uses the Go Alpine image with the same Go version as the project, exposes the port and starts the application. One thing I'd add with more time is more configurability. This is essential for a deployable application and would allow configuring which database is used and where it lives, how files are stored, and defaults such as the maximum limit of latest images.

The frontend is built in two stages: first only the web app, which is then copied directly into a Caddy image that also acts as the reverse proxy. This is simpler since the frontend is just a static page, so I don't have to handle the complexity of routing everything through a separate Caddy service. Caddy can just serve the static files.

I would have liked to also add a DB service (likely Postgres), as this would be closer to a production-grade application.

# Things I'd like to add
- Websocket hub to stream new images live to clients
- Tag filtering with an efficient SQL query using joins
- Cursor param on `/latest` for infinite scrolling and pagination
- Ring buffer for the latest n images (see `./decisions.md` for details)
- Use volumes for the backend
- Replace SQLite with Postgres
- Allow configuration of backend params through config file and/or env vars
- Allow configuration of frontend through env vars for host
- Check the uploaded file for security
- Validate the actual type of the image instead of relying on the header
- MORE TESTS. Smoke tests, fuzz tests, unit tests, end-to-end tests.
- Wrap more errors in `APIError`
- Make conditional error checks and act accordingly using `errors.Is` and `errors.As`
- Add more documentation to the code on functions, structs etc.
- Rate limiting middleware
- More verbose logging containing response status and possibly the request body
