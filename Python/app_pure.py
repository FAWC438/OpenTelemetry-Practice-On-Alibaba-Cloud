import requests
from flask import Flask

app = Flask(__name__)


@app.route('/')
@app.route('/test')
def hello_world():  # put application's code here
    try:
        r = requests.get('http://localhost:5638/test')
        # print(r.text)
        return 'Hello World, this is Python Flask server ! The information {0} is from Java Spring server.'.format(
            r.text)
    except requests.exceptions.InvalidSchema as e:
        return 'Hello World, this is Python Flask server !'


if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000)
