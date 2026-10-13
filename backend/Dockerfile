# --- Build ---
FROM python:3.13.7-slim-bookworm AS build

# install uv
COPY --from=ghcr.io/astral-sh/uv:0.11.24 /uv /usr/local/bin/uv

ENV UV_COMPILE_BYTECODE=1
ENV UV_LINK_MODE=copy

WORKDIR /app

# copy lock files first for better layer caching
COPY pyproject.toml uv.lock ./

# install dependencies without the project
RUN uv sync --frozen --no-dev --no-install-project

# copy application code
COPY src ./src
COPY scripts/init ./

# copy the migrations, so the same image can migrate the database
COPY alembic.ini ./
COPY migrations ./migrations

# install the project itself
RUN uv sync --frozen --no-dev


# --- Production ---
FROM python:3.13.7-slim-bookworm

# locale and timezone
ENV LC_ALL=C.UTF-8
ENV LANG=C.UTF-8
ENV TZ=UTC
ENV PATH="/app/.venv/bin:$PATH"

# install runtime dependencies
RUN apt-get update \
 && apt-get install -y --no-install-recommends curl \
 && apt-get clean \
 && rm -rf /var/lib/apt/lists/*

# create a non-root user to run the application
RUN groupadd --gid 10001 app \
 && useradd --uid 10001 --gid app --create-home app

# set the working directory
WORKDIR /app

# copy built application from build stage
COPY --from=build --chown=app:app /app /app

# switch to the non-root user
USER app

# expose the port that the application will run on
EXPOSE 8000

# run app
CMD ["/app/init"]
