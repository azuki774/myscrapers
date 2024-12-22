FROM python:3.13-bookworm

# Required Packages
RUN apt-get update && \
    apt-get install -y \
    curl \
    unzip \
    wget \
    unzip \
    libglib2.0-0 \
    libnss3 \
    libgconf-2-4 \
    libfontconfig1 && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Install Chrome
RUN wget -q -O - https://dl-ssl.google.com/linux/linux_signing_key.pub | apt-key add -
RUN echo "deb http://dl.google.com/linux/chrome/deb/ stable main" >> /etc/apt/sources.list.d/google.list
RUN apt-get update && apt-get install -y google-chrome-stable && apt-get clean && rm -rf /var/lib/apt/lists/*

# Install driver (Ref. https://sleepless-se.net/2024/03/19/python-selenium-docker/)
RUN wget -O chrome.json https://googlechromelabs.github.io/chrome-for-testing/known-good-versions-with-downloads.json && \
    LINUX_STABLE_URL=$(grep -oP '"url":".*?(?=")' chrome.json | grep 'linux64' | head -n 1 | cut -d'"' -f4) && \
    wget -O chrome.zip $LINUX_STABLE_URL && \
    unzip chrome.zip && \
    rm chrome.zip chrome.json

# AWS Setup
RUN curl -o /var/tmp/awscli.zip https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip && \
    unzip -d /usr/local/bin/ /var/tmp/awscli.zip

# Install Python module
COPY /src/moneyforward/requirements.txt /tmp/
RUN pip install --upgrade pip && pip install -r /tmp/requirements.txt && mkdir -p /data
COPY --chmod=755 build/moneyforward/main.sh /src/main.sh
COPY src/moneyforward/ /src/

ENTRYPOINT ["/src/main.sh"]
