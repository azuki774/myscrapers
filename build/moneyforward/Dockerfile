FROM python:3.9-bookworm

COPY . /app/
WORKDIR /app

# Required Packages
RUN apt-get update && \
    apt-get install -y curl unzip && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# AWS Setup
RUN curl -o /var/tmp/awscli.zip https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip && \
    unzip -d /usr/local/bin/ /var/tmp/awscli.zip

COPY /src/moneyforward/requirements.txt /tmp/
RUN pip install --upgrade pip && pip install -r /tmp/requirements.txt
COPY --chmod=755 build/moneyforward/main.sh /src/main.sh
COPY src/moneyforward/ /src/

ENTRYPOINT ["/src/main.sh"]
