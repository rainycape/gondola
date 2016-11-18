var XMLHttpFactories = [
    function () {return new XMLHttpRequest()},
    function () {return new ActiveXObject("Msxml2.XMLHTTP")},
    function () {return new ActiveXObject("Msxml3.XMLHTTP")},
    function () {return new ActiveXObject("Microsoft.XMLHTTP")}
];

function createXMLHTTPObject() {
    var xmlhttp = null;
    for (var ii = 0; ii < XMLHttpFactories.length; ii++) {
        try {
            xmlhttp = XMLHttpFactories[ii]();
        }
        catch (e) {
            continue;
        }
        break;
    }
    return xmlhttp;
}

function sendRequest(url, data, callback) {
    var req = createXMLHTTPObject();
    if (!req) {
        return;
    }
    var method = data ? "POST" : "GET";
    req.open(method, url, true);
    if (data) {
        req.setRequestHeader('Content-type','application/x-www-form-urlencoded')
    }
    req.onreadystatechange = function () {
        if (req.readyState == 4) {
            callback(req);
        }
    }
    if (req.readyState != 4) {
        req.send(data);
    }
}

function parseJson(text) {
    if (JSON && JSON.parse) {
        return JSON.parse(text);
    }
    return eval('(' + text + ')');
}

function appStatus(callback) {
    sendRequest(GONDOLA_DEV_SERVER_STATUS, null, function (req) {
        var reload = false;
        if (req.status == 404) {
            callback(null);
        } else {
            var resp = parseJson(req.responseText);
            callback(resp);
        }
    });
}