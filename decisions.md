# Frontend
## vite setup decisions
- React compiler since it's now stable and handles memoization plus enforcements of some react rules.
- EsLint since it's still more common

# Backend
- uploading MUST NOT block. The backend shoudl handle multiple uploads at the same time
## Rate Limiting
- fetching the feed will be read heavy so rate limiting this will be usefull. Caching will also help with this

## Caching
- Since this will be a rolling feed (newest images on top oldest at the bottom) i will conider a ring cache where to old entries will just be overriten.
- I will not have to condider TTL with this. It's not important how old the images are. I only care about the latest N images. so overwriting is handles this.
- Constant storage size. Signle heap allocation. 
- naively just cache the relevant entities. Will have to check how caching image will instead will benefit.

## Errors
- Errors MUST NOT leak internal details to api consumer
- Assuming logs are secure and only viewable by trusted people, so interal details can be printed there

## Endpoints
- `/uploads` - accepting uploads as specified by test document.
- `/latest?limit=n` - serves the latest N images

## Storage
- Files will be stored locally in a `file-storage` directory and named with a UUID. 
- Images table will reference the file. Still have to think about only referencing by ID, or by full path.
### DB
- For initial development i will use SQLite for fast iteration and implementaion.
- Later on i can switch to Postgres running in docker.
- Why not MongoDB / NoSQL? Few well defined things have to be stored in the DB + im more familiar with Relational DBs

# Project Structure
- Copy `.pdf` into `.md` since Zed can view `.md`
