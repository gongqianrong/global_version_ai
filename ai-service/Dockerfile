FROM python:3.12-slim
WORKDIR /app
COPY setup.py .
COPY app/ app/
RUN pip install --no-cache-dir .
EXPOSE 8000
CMD ["python", "-c", "print('AI Service ready - install FastAPI to serve HTTP')"]
