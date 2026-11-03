# ai-vision-service — light image by default (stub pipeline, no torch).
# For the real pipeline: docker build --build-arg WITH_FLORENCE=true .
FROM python:3.11-slim

# Tesseract + language packs: v1.0 uses por+eng; the extra packs already
# cover the v2.0 locales (ind, tha, vie, chi_tra) so the image is future-proof.
RUN apt-get update && apt-get install -y --no-install-recommends \
        tesseract-ocr \
        tesseract-ocr-por \
        tesseract-ocr-eng \
        tesseract-ocr-ind \
        tesseract-ocr-tha \
        tesseract-ocr-vie \
        tesseract-ocr-chi-tra \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY requirements.txt requirements-florence.txt ./
RUN pip install --no-cache-dir -r requirements.txt

ARG WITH_FLORENCE=false
RUN if [ "$WITH_FLORENCE" = "true" ]; then \
        pip install --no-cache-dir -r requirements-florence.txt; \
    fi

COPY app ./app

ENV PIPELINE=stub \
    EMBEDDER=stub \
    TESSERACT_LANGS=por+eng+ind+tha+vie+chi_tra \
    PYTHONUNBUFFERED=1

EXPOSE 8083
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8083"]
