const MeasurementTable = (function() {
    const currentPeople = []
    let apiEndpoint = ''
    let tableId = ''
    let headerId = ''

    function getHeatmapClass(deviation) {
        const intensity = Math.abs(deviation);
        if (deviation > 0) {
            if (intensity < 0.5) return 'heatmap-green-light';
            if (intensity < 1) return 'heatmap-green-medium';
            return 'heatmap-green-strong';
        } else {
            if (intensity < 0.5) return 'heatmap-red-light';
            if (intensity < 1) return 'heatmap-red-medium';
            return 'heatmap-red-strong';
        }
    }

    function addPerson(personId) {
        const index = currentPeople.indexOf(personId);
        if (index == -1) {
            currentPeople.push(personId)
        }
    }

    function removePerson(personId) {
        const index = currentPeople.indexOf(personId);
        if (index > -1) {
            currentPeople.splice(index, 1);
        }
    }

    async function updateTable() {
        if (!currentPeople) return;

        const query = currentPeople.map(id => `ids=${id}`).join('&');
        const response = await fetch(`${apiEndpoint}?${query}`);
        const data = await response.json();

        // Update the table
        const headerRow = document.getElementById(headerId);
        const tbody = document.getElementById(tableId).querySelector('tbody');
        tbody.innerHTML = ''; // Clear existing rows

        // Add headers
        headerRow.innerHTML = '<th>Age (years)</th><th>Average (inches)</th>';
        Object.entries(data.People).forEach(([index, person]) => {
            const th = document.createElement('th');
            th.textContent = person.Name;
            headerRow.appendChild(th);
        });

        // Populate rows
        data.Milestones.forEach((milestone) => {
            const row = document.createElement('tr');
            const ageCell = document.createElement('td');

            if (milestone.MilestoneAge === 0) {
                ageCell.textContent = "birth";
            } else if (milestone.MilestoneAge < 1) {
                ageCell.textContent = milestone.MilestoneAge * 12 + " months";
            } else {
                ageCell.textContent = milestone.MilestoneAge + " year";
            }
            row.appendChild(ageCell);

            const averageCell = document.createElement('td');
            averageCell.textContent = milestone.Average.toFixed(2)
            row.appendChild(averageCell);

            Object.entries(milestone.Values).forEach(([index, milestoneValue]) => {
                const heightCell = document.createElement('td');
                heightCell.classList.add('heatmap');
                if (parseFloat(milestoneValue) === 0) {
                    heightCell.textContent = "-"
                } else {
                    const deviation = (parseFloat(milestoneValue) - milestone.Average).toFixed(2)
                    heightCell.textContent = deviation > 0 ? `+${deviation}` : deviation
                    heightCell.classList.add(getHeatmapClass(deviation));
                    heightCell.title = parseFloat(milestoneValue).toFixed(2) + '"'
                }

                row.appendChild(heightCell);
            });

            tbody.appendChild(row);
        });
    }
    return {
        // setup
        setApiEndpoint: (endpoint) => apiEndpoint = endpoint,
        setTableId: (idValue) => tableId = idValue,
        setHeaderId: (idValue) => headerId = idValue,

        // usage
        updateTable: updateTable,
        addPerson: addPerson,
        removePerson: removePerson,
    };
})();