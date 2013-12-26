var COUNT =  300; // 150 seconds of data
var INTERVAL = 1000;

function graphSeries(element) {
    return element.getAttribute('data-plot').split(',');
}

function initializeGraphs() {
    var elements = document.getElementsByClassName('chart')
    var graphs = [];
    var palette = new Rickshaw.Color.Palette();
    for (var ii = 0; ii < elements.length; ii++) {
        var el = elements[ii];
        var axis = document.createElement('div');
        axis.setAttribute('class', 'axis');
        el.appendChild(axis);
        var canvas = document.createElement('div');
        canvas.setAttribute('class', 'canvas');
        el.appendChild(canvas);
        var series = [];
        var s = graphSeries(el);
        for (var jj = 0; jj < s.length; jj++) {
            series.push({name: s[jj]});
        }
        var g = new Rickshaw.Graph({
            element: canvas,
            width: el.offsetWidth,
            height: el.offsetHeight,
            renderer: 'area',
            stroke: true,
            series: new Rickshaw.Series.FixedDuration(series, palette, {
                timeInterval: INTERVAL,
                maxDataPoints: COUNT,
                timeBase: new Date().getTime() / 1000
            })
        });
        palette.color();
        g.series.palette = palette;
        var yaxis = new Rickshaw.Graph.Axis.Y({
            graph: g,
            tickFormat: Rickshaw.Fixtures.Number.formatKMBT,
            element: axis
        });
        yaxis.render();
        graphs[ii] = g;
    }
    setInterval(function () {
        sendRequest('/_gondola_monitor_api', null, function(req) {
            var data = parseJson(req.responseText);
            for (var ii = 0; ii < graphs.length; ii++) {
                var graph = graphs[ii];
                var graphData = {};
                var series = graphSeries(elements[ii]);
                for (var jj = 0; jj < series.length; jj++) {
                    var k = series[jj];
                    graphData[k] = getDottedKey(data, k);
                }
                console.log(graphData);
                graph.series.addData(graphData);
                graph.render();
            }
        }, 'json');
    }, INTERVAL);
}

function getDottedKey(data, k) {
    var keys = k.split('.');
    for (var ii = 0; ii < keys.length; ii++) {
        data = data[keys[ii]];
    }
    return data
}

initializeGraphs();
