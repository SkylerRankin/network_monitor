let lastTimestamp = 1742842731;

const elements = {
    legendText: null,
    legendDate: null,
    legendTime: null,
    legendDown: null,
    legendUp: null,
    legendPing: null,

    latestDate: null,
    latestTime: null,
    latestUp: null,
    latestDown: null,
    latestPingCircle: null,
    latestPingText: null,
};

const gmtToTimeZone = {
    "GMT-0400": "EDT",
    "GMT-4":    "EDT",
    "GMT-0500": "EST",
    "GMT-5":    "EST",
    "GMT-0700": "PDT",
    "GMT-7":    "PDT",
    "GMT-0800": "PST",
    "GMT-8":    "PST",
};

const maxDataPoints = 10000;

const getPingValue = x => x ? 0.2 : 0;

let chart;

const chartOptions = {
    width: 500,
    height: 250,
    scales: {
        "mbps": {
            auto: false,
            range: [0, 1000],
        },
        "boolean": {
            auto: false,
            range: [0, 1],
        },
        x: {
            auto: true,
            visible: false
        }
    },
    axes: [
        { show: false },
        {
            scale: "mbps",
            side: 1,
            values: (_, ticks) => ticks.map(rawValue => `${rawValue} Mbps`),
            size(self, values, axisIdx, cycleNum) {
                let axis = self.axes[axisIdx];

                // bail out, force convergence
                if (cycleNum > 1)
                    return axis._size;

                let axisSize = axis.ticks.size + axis.gap;

                // find longest value
                let longestVal = (values ?? []).reduce((acc, val) => (
                    val.length > acc.length ? val : acc
                ), "");

                if (longestVal != "") {
                    self.ctx.font = axis.font[0];
                    axisSize += self.ctx.measureText(longestVal).width / devicePixelRatio;
                }
                return Math.ceil(axisSize);
            },
            font: "12px monospace",
        },
        {
            scale: "boolean",
            show: false,
            side: 3
        }
    ],
    series: [
        {},
        {
            show: true,
            spanGaps: true,
            scale: "mbps",
            stroke: "green",
            fill: "#4caf505e",
            points: { show: false }
        },
        {
            show: true,
            spanGaps: false,
            scale: "mbps",
            stroke: "blue",
            fill: "#4c7daf5e",
            points: { show: false }
        },
        {
            show: true,
            spanGaps: true,
            scale: "boolean",
            stroke: "#8e4fdb",
            points: { show: false }
        }
    ],
    legend: {
        show: false,
    },
    hooks: {
        setSeries: [ (u, seriesIdx) => console.log('setSeries', seriesIdx) ],
        setLegend: [ u => updateLegend(u) ],
    },
    cursor: {
        y: false,
        drag: { setScale: false }
    },
    select: {
        show: false
    },
};

const updateLegend = u => {
    const i = u.legend.idxs;
    if (i[0] === null) {
        elements.legendText.classList.add("hidden");
        return;
    } else if (elements.legendText.classList.contains("hidden")) {
        elements.legendText.classList.remove("hidden");
    }

    const date = new Date(u.data[0][i[0]]);
    elements.legendDate.innerHTML = date.toLocaleDateString();
    elements.legendTime.innerHTML = date.toLocaleTimeString();

    const gmt = date.toString().match(/GMT-\d+/g);
    if (gmt && gmt.length === 1 && gmtToTimeZone[gmt[0]]) {
        elements.legendTime.innerHTML += ` ${gmtToTimeZone[gmt[0]]}`;
    }

    elements.legendDown.innerHTML = Math.floor(u.data[1][i[1]]);
    elements.legendUp.innerHTML = Math.floor(u.data[2][i[2]]);
    elements.legendPing.innerHTML = u.data[3][i[3]] > 0 ? "received" : "failed";
}

const updateLatestSummary = () => {
    const endIndex = chart.data[0].length - 1;
    const timestamp = chart.data[0][endIndex];
    let download = chart.data[1][endIndex];
    let upload = chart.data[2][endIndex];
    const ping = chart.data[3][endIndex] > 0;

    let i = endIndex;
    while (download === undefined && i >= 0) {
        i -= 1;
        download = chart.data[1][i];
    }

    i = endIndex;
    while (upload === undefined && i >= 0) {
        i -= 1;
        upload = chart.data[2][i];
    }

    const date = new Date(timestamp);

    elements.latestDate.innerHTML = date.toLocaleDateString();
    elements.latestTime.innerHTML = date.toLocaleTimeString();
    const gmt = date.toString().match(/GMT-\d+/g);
    if (gmt && gmt.length === 1 && gmtToTimeZone[gmt[0]]) {
        elements.latestTime.innerHTML += ` ${gmtToTimeZone[gmt[0]]}`;
    }

    elements.latestDown.innerHTML = Math.floor(download);
    elements.latestUp.innerHTML = Math.floor(upload);

    if (ping) {
        elements.latestPingCircle.classList.remove("red_dot");
        elements.latestPingCircle.classList.add("green_dot");
        elements.latestPingText.innerHTML = "Success";
    } else {
        elements.latestPingCircle.classList.remove("green_dot");
        elements.latestPingCircle.classList.add("red_dot");
        elements.latestPingText.innerHTML = "Failure";
    }
}

/**
 * Uses undefined to fill in missing values for speed series. Using null adds a gap,
 * which makes this series useless since all data points are surrounded by nulls.
 * https://github.com/leeoniya/uPlot/issues/850#issuecomment-1602969828
 */
const loadInitialData = async () => {
    const res = await fetch("/batch");
    if (res.status !== 200) {
        console.error(`/batch: ${res.status}, ${res.statusText}`);
        return;
    }

    const json = await res.json();
    chart.data[0] = json["timestamps"];
    chart.data[1] = json["download"].map(x => x === 0 ? undefined : x);
    chart.data[2] = json["upload"].map(x => x === 0 ? undefined : x);
    chart.data[3] = json["ping"].map(getPingValue);

    if (chart.data[0].length > maxDataPoints) {
        for (let i = 0; i < chart.data.length; i++) {
            chart.data[i].splice(0, chart.data[0].length - maxDataPoints);
        }
    }

    chart.setData(chart.data);
    updateLatestSummary();
}

const setConnectionStatus = status => {
    const dot = document.getElementById("title_connected_circle");
    const text = document.getElementById("title_active_text");

    if (status === "connected") {
        dot.className = "dot green_dot";
        text.innerHTML = "Connected";
    } else if (status === "not connected") {
        dot.className = "dot red_dot";
        text.innerHTML = "Not connected";
    } else if (status === "connecting") {
        dot.className = "dot yellow_dot";
        text.innerHTML = "Connecting...";
    } else {
        console.error(`setConnectionStatus invalid parameter \"${status}\"`);
    }
}

const connectToWebSocket = () => {
    const socket = new WebSocket("/ws");
    setConnectionStatus("connecting");

    socket.onopen = () => setConnectionStatus("connected");
    socket.onclose = () => setConnectionStatus("not connected");
    socket.onerror = error => console.error(`Websocket connection error: `, error);

    socket.onmessage = event => {
        const info = JSON.parse(event.data);
        chart.data[0].push(info["timestamp"]);
        chart.data[1].push(info["download"] === -1 ? undefined : info["download"]);
        chart.data[2].push(info["upload"] === -1 ? undefined : info["upload"]);
        chart.data[3].push(getPingValue(info["ping"]));

        if (chart.data[0].length > maxDataPoints) {
            for (let i = 0; i < chart.data.length; i++) {
                chart.data[i].splice(0, chart.data[0].length - maxDataPoints);
            }
        }

        chart.setData(chart.data);
        updateLatestSummary();
        console.log(info);
    }
};

window.onload = async () => {
    elements.legendText = document.getElementById("legend_text");
    elements.legendDate = document.getElementById("legend_date");
    elements.legendTime = document.getElementById("legend_time");
    elements.legendDown = document.getElementById("legend_down");
    elements.legendUp = document.getElementById("legend_up");
    elements.legendPing = document.getElementById("legend_ping");

    elements.latestDate = document.getElementById("latest_date");
    elements.latestTime = document.getElementById("latest_time");
    elements.latestUp = document.getElementById("latest_up");
    elements.latestDown = document.getElementById("latest_down");
    elements.latestPingCircle = document.getElementById("latest_ping_circle");
    elements.latestPingText = document.getElementById("latest_ping_text");

    const data = [
        [], // x-values (timestamps)
        [], // y-values (download speed)
        [], // y-values (upload speed)
        [], // y-values (ping success)
    ];
    chart = new uPlot(chartOptions, data, document.getElementById("chart"));

    setConnectionStatus("not connected");
    await loadInitialData();
    connectToWebSocket();
}
