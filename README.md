# Blog

My new blog

## Getting Started

```bash
git clone https://github.com/kevingil/blog
cd blog
pnpm install
```

## Running Locally

Setup your .env

```bash
# Database 
TURSO_CONNECTION_URL=http://127.0.0.1:8080 # dev url, replace with your turso db url
TURSO_AUTH_TOKEN=xxxxxxxxxxxxx
#AI
OPENAI_API_KEY=sk-proj-xxxxxx
#File uploads
S3_BUCKET=xxxxxxxxxxxxx
S3_ACCOUNT_ID=xxxxxxxxxxxxx
S3_ACCESS_KEY_ID=xxxxxxxxxxxxx
S3_ACCESS_KEY_SECRET=xxxxxxxxxxxxx
S3_SESSION_TOKEN=xxxxxxxxxxxxx
S3_ENDPOINT=xxxxxxxxxxxxx
NEXT_PUBLIC_S3_URL_PREFIX=xxxxxxxxxxxxx

```

Then, run the database migrations and seed the database with a default user:

```bash
# For local development
turso dev --db-file local.db

# Setup and seed database

pnpm db:migrate
pnpm db:seed

#Making schema changes
pnpm db:generate
pnpm db:migrate

```

This will create the following user and team:

- User: `admin@admin.com`
- Password: `admin123`

You can, of course, create new users as well through `/sign-up`.

Finally, run the Next.js development server:

```bash
#dev
pnpm dev
#build cmd
pnpm build

```

Open [http://localhost:3000](http://localhost:3000) in your browser to see the app in action.

