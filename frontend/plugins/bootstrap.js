import SfdumpFunc from '../Utils/dumper'

export default function ({store}, inject) {
  window.Sfdump = SfdumpFunc(window.document)

  store.dispatch('theme/detect')
}
