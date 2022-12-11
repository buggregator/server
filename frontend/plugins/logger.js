import { white } from 'console-log-colors'

const prefix = white.bgBlue.bold

export class Logger {
  constructor(mode, prefix = '') {
    this.mode = mode
    this.prefix = prefix
  }

  withPrefix(prefix) {
    return new Logger(this.mode, prefix)
  }

  debug(...content) {
    this.__log('success', ...content)
  }

  error(...content) {
    this.__log('error', ...content)
  }

  info(...content) {
    this.__log('info', ...content)
  }

  __log(type, ...content) {
    if (this.mode !== 'development') {
      return
    }

    if (this.prefix) {
      content.unshift(prefix(`[${this.prefix}]`))
    }

    switch (type) {
      case 'success':
        console.info(...content)
        break
      case 'info':
        console.info(...content)
        break
      case 'error':
        console.error(...content)
        break
    }
  }
}

const logger = (context, inject) => {
  const logger = new Logger(process.env.NODE_ENV)

  inject('logger', logger)
  context.$logger = logger
};

export default logger
