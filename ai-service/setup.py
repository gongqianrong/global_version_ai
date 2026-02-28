from setuptools import setup, find_packages

setup(
    name="rakutao-ai-service",
    version="0.1.0",
    packages=find_packages(),
    install_requires=[
        "fastapi>=0.95.0,<0.100.0",
        "uvicorn>=0.20.0,<0.25.0",
    ],
    extras_require={"dev": ["pytest", "httpx>=0.23.0,<0.25.0"]},
)
