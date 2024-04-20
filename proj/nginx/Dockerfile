FROM nginx:1.25.4

WORKDIR C:/Users/jorge/Documents/feup/4ano/semestre2/GRS/FEUP-GRS/proj/html

COPY . /usr/share/nginx/html/

WORKDIR C:/Users/jorge/Documents/feup/4ano/semestre2/GRS/FEUP-GRS/proj/nginx

#COPY ./nginx.conf /etc/nginx/nginx.conf

EXPOSE 80

CMD [ "nginx", "-g", "daemon off;"]