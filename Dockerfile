FROM golang

COPY . .

RUN ls

CMD [ "echo","bien" ]