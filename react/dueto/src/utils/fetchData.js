function getSessionId() {
    let cookies = document.cookie
    let cookieParts = cookies.split("=")

    if(cookieParts[0] == "SESSIONID") {
      return cookieParts[1]
    } 
    else {
      return ""
    }
}

function encodeUrl(data) {
  const formBody = Object.keys(data)
    .map(
      key =>
        encodeURIComponent(key) +
        '=' +
        encodeURIComponent(data[key])
    )
    .join('&')
  return formBody
}

export const getProfileData = async (name) => {
  let id = getSessionId()

  const data = {
     sessionid: id,
     username: name
  }
  const params = encodeUrl(data)
  const response = await fetch("/api/profile?" + params, { credentials: "include" })
  const newData = await response.json()

  return newData
}

export const getHomeData = async () => {
  let id = getSessionId()

  const data = {
     sessionid: id
  }
  const params = encodeUrl(data)
  const response = await fetch("/api/home?" + params, { credentials: "include" })
  const newData = await response.json()

  return newData
}

export const getDiscoverData = async () => {
  let id = getSessionId()

  const data = {
     sessionid: id
  }
  const params = encodeUrl(data)
  const response = await fetch("/api/discover?" + params, { credentials: "include" })
  const newData = await response.json()

  return newData
}

export const getVideoUrl = (artistId, video) => {
  let id = getSessionId()

  const data = {
     sessionid: id,
     artist: artistId,
     name: video
  }
  const params = encodeUrl(data)

  return "/api/video?" + params
}

export const addVideo = async (feedUrl, video) => {
  const bodyData = {video: video}
  const formBody = encodeUrl(bodyData)

  const response = await fetch(feedUrl, {
    body: formBody,
    credentials: 'include',
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/x-www-form-urlencoded',
      'cache-control': 'no-cache'
    }
  })

  const newData = await response.json()
  return newData
}
