var tag = document.createElement('script');

tag.src = "https://www.youtube.com/iframe_api";
var firstScriptTag = document.getElementsByTagName('script')[0];
firstScriptTag.parentNode.insertBefore(tag, firstScriptTag);

var spotifyDeviceId;
var spotifySongPlaying = false;
var spotifyPlayer;
var player;
var autoPlayEnabled = false;

function onYouTubeIframeAPIReady() {
    player = new YT.Player('player', {
        height: '390',
        width: '640',
        videoId: 'Ssmiv3LvWcE',
        origin: 'https://burtbot.app',
        playerVars: {
            'playsinline':1
        },
        events: {
            'onReady': onPlayerReady,
            'onStateChange': onPlayerStateChange
        }
    });
}

function onPlayerReady(event) {
    event.target.playVideo();
}

var done = false;
function onPlayerStateChange(event) {
    if (event.data == YT.PlayerState.ENDED) {
        console.log('video is done, looking for the next request');
        // get the song id of the next request
        playNextRequest()
    }
}

function stopVideo() {
    player.stopVideo();
}

// load the given json into elements on the page
function showQueue(queue) {
    const ul = document.getElementsByTagName('ul')[0];
    ul.innerHTML = "";
    for (const [index, request] of queue.entries()) {
        var l = document.createElement('li')
        l.innerHTML = `<strong>${request.SongTitle} - ${request.SongArtists[0]}</strong> - ${request.User} - Duration: ${request.Duration} `;
        l.innerHTML += `${request.Added} <a href="https://burtbot.app/play_request?id=${index}">Play Now</a>`;
        l.innerHTML += `<input class="remove-btn" type="button" value="Remove"/>`;
        l.setAttribute('song-id', request.SongID);
        l.setAttribute('service', request.Service);
        l.className = "songRequest";
        const remBtn = l.getElementsByClassName("remove-btn")[0]
        remBtn.addEventListener("click", async (e) => {
            const conf = confirm("Do you want to remove this request?")
            if (conf) {
                const r = await fetch(`https://burtbot.app/remove_request?id=${index}`)
                if (!r.ok) {
                    console.error("Problem deleting request: " + r.status);
                }
                getCurrentQueue();
            }
        });
        const playLink = l.getElementsByTagName('a')[0]
        playLink.addEventListener("click", async (e) => {
            e.preventDefault();
            if (confirm("Play this request now?")) {
               playRequest(index);
            }
        });
        ul.appendChild(l)
    }
}

// Need a polling function to check for additions to the queue
// maybe every 5 seconds or something.
async function getCurrentQueue() {
    const url = "https://burtbot.app/current_queue";
    try {
        const response = await fetch(url);
        if (!response.ok) {
            throw new Error(`Error fetching queue: ${response.status}`);
        }
        const json = await response.json();
        //update the token
        document.getElementById('spotifyToken').value = json.SpotifyToken;
        showQueue(json.CurrentQueue);
        const ytPlayerState = player.getPlayerState();
        if (!spotifySongPlaying && (ytPlayerState == 0 || ytPlayerState == -1) && autoPlayEnabled) {
            console.log("Trying to autoplay new request! I'm sure this is working correctly!");
        }
    } catch (error) {
        console.error(error.message);
    }
}

function pollQueue() {
    getCurrentQueue();
    setTimeout(pollQueue, 5000);
}
pollQueue()

async function playNextRequest() {
    playRequest(0);
}

async function playRequest(index) {
    player.pauseVideo();
    await spotifyPlayer.pause();
    playingSpotifySong = false;
    var requests = document.getElementsByClassName('songRequest');
    if (requests.length == 0) {
        console.log('all out of request, sad');
        return;
    }
    const service = requests[index].getAttribute('service');
    if (service == "Youtube") {
        player.loadVideoById(requests[index].getAttribute('song-id'));
    } else if (service == "Spotify") {
        const response = await fetch(`https://api.spotify.com/v1/me/player/play?device_id=${spotifyDeviceId}`, {
            method: 'PUT',
            headers: {
                Authorization: 'Bearer ' + document.getElementById('spotifyToken').value
            },
            body: JSON.stringify({"uris": [`spotify:track:${requests[index].getAttribute('song-id')}`]})
        });
        if (!response.ok) {
            const json = await response.json()
            console.error(`Eror playing spotify track: ${json.error.status} - ${json.error.message}`);
            return;
        }
    } else {
        return;
    }
    const setPlayingURL = requests[index].getElementsByTagName('a')[0].getAttribute('href');
    const resp = await fetch(setPlayingURL);
    if (!resp.ok) {
        console.error("Couldn't mark song playing on bot: " + resp.status);
    }
    getCurrentQueue();
}

window.onSpotifyWebPlaybackSDKReady = () => {
    const player = new Spotify.Player({
        name: 'BurtBot Request Queue Manager',
        getOAuthToken: cb => { cb(document.getElementById('spotifyToken').value); },
        volume: 0.5
    });
    player.addListener('ready', ({ device_id}) => {
        spotifyDeviceId = device_id;
        console.log(`Ready with Device ID ${device_id}`);
    });
    player.addListener('not_ready', ({ device_id }) => {
        console.log(`Device ID has gone offline ${device_id}`);
    });
    player.addListener('initialization_error', ({ message }) => {
        console.error(message);
    });
    player.addListener('authentication_error', ({ message }) => {
        console.error(message);
    });
    player.addListener('account_error', ({ message }) => {
        console.error(message);
    });
    player.addListener('player_state_changed', ({
        position,
        paused,
        duration,
        track_window: { current_track }
    }) => {
        if (!paused) {
            spotifySongPlaying = true;
        }
        if (position == 0 && paused && spotifySongPlaying) {
            spotifySongPlaying = false;
            playNextRequest();
        }
    });
    document.getElementById('togglePlay').onclick = function() {
        player.togglePlay();
    };
    player.connect();
    spotifyPlayer = player;
}

document.getElementById('autoplay-cb').addEventListener('change', (e) => {
    autoPlayEnabled = e.target.checked;
});
