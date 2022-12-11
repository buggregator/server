const showEvent = ({$axios}) => async (uuid) => $axios.$get(`api/event/${uuid}`)
  .then((response) => response)

// const showEvents = ({$ws}) => async (type) => $ws.rpc(`get:api/events`, {type})
//   .then((response) => response.data.data)

const showEvents =  ({$axios}) => async (name, address) => $axios.$get('/api/events')
  .then((response) => response.data)

const clearEvents = ({$ws}) => async (type) => $ws.rpc(`delete:api/events`, {type})

const deleteEvent = ({$ws}) => async (uuid) => $ws.rpc(`delete:api/event/${uuid}`)

export default {
  showEvent,
  showEvents,
  clearEvents,
  deleteEvent
}
