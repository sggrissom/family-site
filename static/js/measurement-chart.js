const LineChart = (function() {
    const chartData = {
        datasets: []
    };
    let lineChart;
    let apiEndpoint;
    let dataFormatter;
    let xTitle;
    let yTitle;
    let xTooltipCallback;
    let yTooltipCallback;
    let lastValidZoom = {
        x: { min: null, max: null },
        y: { min: null, max: null }
    };

    const initializeChart = (chartCanvasId) => {
        const ctx = document.getElementById(chartCanvasId).getContext('2d');
        lineChart = new Chart(ctx, {
            type: 'line',
            data: chartData,
            options: {
                responsive: true,
                parsing: false,
                interaction: {
                    mode: 'nearest',
                    axis: 'x',
                    intersect: true,
                },
                plugins: {
                    tooltip: {
                        callbacks: {
                            title: xTooltipCallback,
                            label: yTooltipCallback,
                        }
                    },
                    zoom: {
                        pan: {
                            enabled: true,
                            mode: 'xy',
                        },
                        limits: {
                            x: {min: 0, max: 15},
                            y: {min: 0, max: 80},
                        },
                        zoom: {
                            wheel: { enabled: true },
                            drag: { enabled: true },
                            pinch: { enabled: true },
                            mode: 'xy',
                            onZoomComplete({chart}) {
                                if (!isZoomValid(chart)) {
                                    chart.options.scales.x.min = lastValidZoom.x.min;
                                    chart.options.scales.x.max = lastValidZoom.x.max;
                                    chart.options.scales.y.min = lastValidZoom.y.min;
                                    chart.options.scales.y.max = lastValidZoom.y.max;
                                    chart.tooltip.setActiveElements([], {x: 0, y: 0});
                                    chart.update();
                                } else {
                                    lastValidZoom.x.min = chart.scales.x.min;
                                    lastValidZoom.x.max = chart.scales.x.max;
                                    lastValidZoom.y.min = chart.scales.y.min;
                                    lastValidZoom.y.max = chart.scales.y.max;
                                    chart.tooltip.setActiveElements([], {x: 0, y: 0});
                                    chart.update();
                                }
                            }
                        }
                    }
                },
                scales: {
                    x: {
                        type: 'linear',
                        title: {
                            display: true,
                            text: xTitle,
                        },
                        ticks: {
                            callback: function(value) {
                                return value.toFixed(1);
                            }
                        }
                    },
                    y: {
                        title: {
                            display: true,
                            text: yTitle,
                        },
                    },
                },
            },
        });
        
        updateZoomLimits();
    };

    const isZoomValid = (chart) => {
        const xMin = chart.scales.x.min;
        const xMax = chart.scales.x.max;
        const yMin = chart.scales.y.min;
        const yMax = chart.scales.y.max;
        for (let dataset of chart.data.datasets) {
            for (let point of dataset.data) {
                if (point.x >= xMin && point.x <= xMax && point.y >= yMin && point.y <= yMax) {
                    return true;
                }
            }
        }
        return false;
    };

    const updateZoomLimits = () => {
        let xMin = Infinity, xMax = -Infinity, yMin = Infinity, yMax = -Infinity;
        chartData.datasets.forEach(dataset => {
            dataset.data.forEach(point => {
                if (point.x < xMin) xMin = point.x;
                if (point.x > xMax) xMax = point.x;
                if (point.y < yMin) yMin = point.y;
                if (point.y > yMax) yMax = point.y;
            });
        });
        if (xMin === Infinity) {
            xMin = 0; xMax = 15; yMin = 0; yMax = 80;
        }
        lineChart.options.plugins.zoom.limits.x = { min: xMin, max: xMax };
        lineChart.options.plugins.zoom.limits.y = { min: yMin, max: yMax };
        lastValidZoom.x.min = xMin;
        lastValidZoom.x.max = xMax;
        lastValidZoom.y.min = yMin;
        lastValidZoom.y.max = yMax;
        lineChart.update();
    };

    // Fetch and add a person's data
    const addPerson = async (personId) => {
        if (!personId) {
            alert('Invalid ID');
            return;
        }
        if (chartData.datasets.find(ds => ds.label.includes(`ID:${personId}`))) {
            alert('Person already added');
            return;
        }
        try {
            const response = await fetch(`${apiEndpoint}/${personId}`);
            if (!response.ok) {
                alert(`Error fetching data for ID ${personId}`);
                return;
            }
            const data = await response.json();
            if (data == null) {
                alert(`Error fetching data for ID ${personId}`);
                return;
            }
            const dataset = {
                label: `${data[0].PersonName} (ID:${personId})`,
                data: dataFormatter(data),
                borderColor: getIndexedColor(chartData.datasets.length),
                backgroundColor: 'rgba(0, 0, 0, 0)',
                fill: false,
                tension: 0.1,
            };
            chartData.datasets.push(dataset);
            lineChart.update();
            updateZoomLimits();
        } catch (error) {
            console.error(error);
            alert(error.message);
        }
    };

    // Remove a person's data
    const removePerson = (personId) => {
        if (!personId) return;
        const datasetIndex = chartData.datasets.findIndex(ds => ds.label.includes(`ID:${personId}`));
        if (datasetIndex === -1) {
            alert(`No data found for ID ${personId}`);
            return;
        }
        chartData.datasets.splice(datasetIndex, 1);
        lineChart.update();
        updateZoomLimits();
    };

    function getIndexedColor(index) {
        const colors = [
            "#39e3cf",
            "#d048fa",
            "#fa48b0",
            "#f1fa48",
            "#fa9248",
        ];
        return colors[index] ?? "#00000";
    }

    const resetZoom = () => {
        lineChart.resetZoom();
        updateZoomLimits();
    };

    return {
        setApiEndpoint: (newApiEndpoint) => apiEndpoint = newApiEndpoint,
        setDataFormatter: (newDataFormatter) => dataFormatter = newDataFormatter,
        setXTitle: (newXTitle) => xTitle = newXTitle,
        setYTitle: (newYTitle) => yTitle = newYTitle,
        setXTooltipCallback: (newXTooltipCallback) => xTooltipCallback = newXTooltipCallback,
        setYTooltipCallback: (newYTooltipCallback) => yTooltipCallback = newYTooltipCallback,
        initializeChart: initializeChart,
        addPerson: addPerson,
        removePerson: removePerson,
        resetZoom: resetZoom
    };
})();