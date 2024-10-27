# Blog

My blog




### Development

```bash
# start test databases
docker compose up -d

# From (env) server/
pip install -r requirements.txt
gunicorn -w 4 -b 0.0.0.0:5000 app.wsgi:app

## From client/
pnpm dev / start # start dev server and open browser
pnpm build # build for production
pnpm preview # locally preview production build
pnpm test # launch test runner

```
