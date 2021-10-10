docker build -t doremy .

docker run -d \
    --name doremy \
    -v doremy-vol:/doremy \
    doremy
