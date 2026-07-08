# Container image for the Deeb RDS/Aurora onboarding sweep.
# Runs as an EKS CronJob (one per AWS account), in-VPC with network access to RDS.
FROM python:3.12-slim

# psycopg2-binary and pymysql are wheels; no build toolchain needed.
WORKDIR /app

COPY requirements-aws.txt ./
RUN pip install --no-cache-dir -r requirements-aws.txt

COPY discover.py aws-onboard-to-deeb.py ./

# Run as a non-root user.
RUN useradd --system --uid 10001 onboarder
USER 10001

ENTRYPOINT ["python", "aws-onboard-to-deeb.py"]
