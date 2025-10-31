# Use Amazon Linux as base image
FROM amazonlinux:latest

# Install dependencies
RUN yum update -y && yum install -y make curl unzip tar git && yum clean all

# Install Python (latest) and Pip
RUN yum install -y python3 && ln -s /usr/bin/python3 /usr/bin/python

# Install Go (latest)
RUN curl -OL https://golang.org/dl/go1.21.0.linux-amd64.tar.gz && tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz && rm go1.21.0.linux-amd64.tar.gz

ENV PATH="/usr/local/go/bin:${PATH}"

# Install AWS CLI
RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" && unzip awscliv2.zip && ./aws/install && rm -rf awscliv2.zip aws

# Install YAML Lint
RUN pip3 install yamllint

# Set work directory inside the container
WORKDIR /app

# Copy project files into the container
COPY . .

# Default command
CMD ["/bin/bash"]
