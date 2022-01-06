// Initialize the map
const POLYGON_LIMIT = 10000;
let map = L.map('map').setView([61.1089065, -149.720477], 10);
L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
}).addTo(map);
let statusField = document.getElementById('status');

// Displaying fetch bounds
let currentFetchBounds = L.polygon([], { color: 'rgba(0,0,0,0.2)', fill: false });
currentFetchBounds.addTo(map);
let polygons = L.layerGroup().addTo(map);

// Helper functions
let geoJsonCoordinatesToLatLngs = (coordinates) => {
    let points = coordinates[0];
    points = points.map(point => [point[1], point[0]]);
    return points;
}

let updateFetchBounds = () => {
    let bounds = map.getBounds().pad(-0.2);
    let geojson = L.rectangle(bounds).toGeoJSON();
    let points = geoJsonCoordinatesToLatLngs(geojson.geometry.coordinates);
    currentFetchBounds.setLatLngs(points);
    return geojson.geometry.coordinates;
}

let updateStatusMessage = (message) => {
    statusField.innerText = message;
}

let hidePolygons = () => {
    polygons.clearLayers();
}

let displayError = (error) => {
    updateStatusMessage('Error occurred, see console for details.');
    console.log(error);
}

let displayPolygons = (result) => {
    updateStatusMessage(`Found ${result.polygons.length} polygons.`);
    polygons.clearLayers();
    result.polygons.forEach(element => {
        polygons.addLayer(L.polygon(geoJsonCoordinatesToLatLngs(element.body), { color: 'yellow' }));
    });
}

let fetchPolygonBodies = async (result) => {
    if (result.ids.length > 0) {
        if (result.ids.length <= POLYGON_LIMIT) {
            fetch('/api/polygons',
                {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json;charset=utf-8'
                    },
                    body: JSON.stringify({
                        ids: result.ids
                    })
                }).then(async (response) => {
                    if (!response.ok) {
                        displayError(response);
                        return;
                    }
                    try {
                        let result = await response.json();
                        displayPolygons(result);
                    } catch (e) {
                        displayError(e);
                    }
                });
        } else {
            updateStatusMessage(`Too many results (${result.ids.length} > ${POLYGON_LIMIT}), please zoom in.`);
            hidePolygons();
        }
    } else {
        updateStatusMessage('No results.');
        hidePolygons();
    }
}

let fetchPolygonIDs = () => {
    updateStatusMessage('Fetching...');
    let fetchBounds = updateFetchBounds();
    // Rewind the coordinates to create CCW loop.
    fetchBounds[0] = fetchBounds[0].reverse();
    fetch('/api/search/polygons/by_polygon',
        {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json;charset=utf-8'
            },
            body: JSON.stringify({
                polygon: fetchBounds
            })
        })
        .then(async (response) => {
            if (!response.ok) {
                displayError(response);
                return;
            }
            try {
                let result = await response.json();
                fetchPolygonBodies(result);
            } catch (e) {
                displayError(e);
            }
        });
}

map.on('moveend', function (e) {
    fetchPolygonIDs();
});
