window.onload = function () {
var app = new Vue({
    el: '#app',

    data: {
        ws: null, // Our websocket
        captchaData: []
    },

    created: function() {
        var self = this;
        this.ws = new WebSocket('ws://' + window.location.host + '/ws');
        this.ws.addEventListener('message', function(e) {
            var captcha = JSON.parse(e.data);
            self.captchaData.push(captcha);
        });
    },

    methods: {
        send: function (md5, answer) {
            this.captchaData = this.captchaData.filter(c => c.Md5 != md5);
            this.ws.send(
                JSON.stringify({
                    md5: md5,
                    answer: answer
                }
            ));
        },
        answer1: function (answer) {
            return answer.substr(0,4);
        },
        answer2: function (answer) {
            return answer.substr(4, 4);
        },
        answer3: function (answer) {
            return answer.substr(8,4);
        },
        answer4: function (answer) {
            return answer.substr(12,4);
        },
    }
})
}