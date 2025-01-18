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
                    intersect: false
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
                            wheel: {
                                enabled: true,
                            },
                            drag: {
                                enabled: true,
                            },
                            pinch: {
                                enabled: true
                            },
                            mode: 'xy',
                        }
                    }
                },
                scales: {
                    x: {
                        type: 'linear', // Ensure x-axis is treated as numeric
                        title: {
                            display: true,
                            text: xTitle,
                        },
                        ticks: {
                            callback: function(value) {
                                return value.toFixed(1)
                            }
                        }
                    },
                    y: {
                        title: {
                            display: true,
                            text: yTitle,
                        },
                        min: undefined,
                        max: undefined,
                    },
                },
            },
        });
    }


    // Function to fetch and add a person's data
    const addPerson = async (personId) => {
        if (!personId) {
            alert('invalid id');
            return;
        };

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
                borderColor: getRandomColor(),
                backgroundColor: 'rgba(0, 0, 0, 0)',
                fill: false,
                tension: 0.1,
            };

            chartData.datasets.push(dataset);
            lineChart.update();
        } catch (error) {
            console.error(error);
            alert(error.message)
        }
    }

    // Function to remove a person's data
    const removePerson = (personId) => {
        if (!personId) return;

        const datasetIndex = chartData.datasets.findIndex(ds => ds.label.includes(`ID:${personId}`));
        if (datasetIndex === -1) {
            alert(`No data found for ID ${personId}`);
            return;
        }

        // Remove dataset and update chart
        chartData.datasets.splice(datasetIndex, 1);
        lineChart.update();
    }

    // Utility function to generate random colors
    function getRandomColor() {
        const letters = '0123456789ABCDEF';
        let color = '#';
        for (let i = 0; i < 6; i++) {
            color += letters[Math.floor(Math.random() * 16)];
        }
        return color;
    }

    return {
        //initializers
        setApiEndpoint: (newApiEndpoint) => apiEndpoint = newApiEndpoint,
        setDataFormatter: (newDataFormatter) => dataFormatter = newDataFormatter,
        setXTitle: (newXTitle) => xTitle = newXTitle,
        setYTitle: (newYTitle) => yTitle = newYTitle,
        setXTooltipCallback: (newXTooltipCallback) => xTooltipCallback = newXTooltipCallback,
        setYTooltipCallback: (newYTooltipCallback) => yTooltipCallback = newYTooltipCallback,
        
        //create chart
        initializeChart: initializeChart,

        //usage
        addPerson: addPerson,
        removePerson: removePerson,
    };
})();
