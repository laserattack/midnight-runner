function sendJSON(jsonData, endpoint) {
    return fetch(endpoint, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(jsonData)
    })
    .then(response => {
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
    });
}

function receiveJSON(endpoint) {
    return fetch(endpoint)
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            return response.json();
        });
}

function formatTimestamp(timestamp) {
    if (!timestamp) return 'Unknown';
    const date = new Date(timestamp * 1000);
    return date.toLocaleString('ru-RU', {
        day: '2-digit',
        month: '2-digit',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
    });
}

function updateJobsTable(data) {
    const jobs = data.jobs || {};
    const jobsArray = Object.entries(jobs);

    document.getElementById('totalJobs')
        .textContent = jobsArray.length;

    const tableBody = document.getElementById('jobsTableBody');
    tableBody.innerHTML = '';

    if (jobsArray.length === 0) {
        document.getElementById('error').style.display = 'block';
        document.getElementById('error').innerHTML =
            '<p>No jobs configured</p>';
        document.getElementById('content').style.display = 'none';
        return;
    }

    jobsArray.forEach(([name, job]) => {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td class="job-name">${name}</td>
            <td>${job.type}</td>
            <td>${job.description}</td>
            <td><code>${job.config.command}</code></td>
            <td><code>${job.config.cron_expression}</code></td>
            <td>${job.config.status}</td>
            <td>${job.config.timeout}</td>
            <td>${job.config.max_retries}</td>
            <td>${job.config.retry_interval}</td>
        `;
        tableBody.appendChild(row);
    });

    document.getElementById('content').style.display = 'block';
    document.getElementById('error').style.display = 'none';
}

