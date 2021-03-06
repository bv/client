// @flow
import * as PushNotifications from 'react-native-push-notification'
import {PushNotificationIOS} from 'react-native'
import * as PushConstants from '../constants/push'
import {eventChannel} from 'redux-saga'

function requestPushPermissions (): Promise<*> {
  return PushNotifications.requestPermissions()
}

function showMainWindow () {
  return () => {
    // nothing
  }
}

function configurePush () {
  return eventChannel(emitter => {
    PushNotifications.configure({
      onRegister: (token) => {
        let tokenType: ?PushConstants.TokenType
        switch (token.os) {
          case 'ios': tokenType = PushConstants.tokenTypeApple; break
          case 'android': tokenType = PushConstants.tokenTypeAndroidPlay; break
        }
        if (tokenType) {
          emitter(({
            payload: {
              token: token.token,
              tokenType,
            },
            type: 'push:pushToken',
          }: PushConstants.PushTokenAction))
        } else {
          emitter(({
            payload: {
              error: new Error('Unrecognized os for token:', token),
            },
            type: 'push:registrationError',
          }: PushConstants.PushRegistrationError))
        }
      },
      senderID: PushConstants.androidSenderID,
      onNotification: (notification) => {
        emitter(({
          payload: notification,
          type: 'push:notification',
        }: PushConstants.PushNotificationAction))
      },
      onError: (error) => {
        emitter(({
          payload: {error},
          type: 'push:error',
        }: PushConstants.PushError))
      },
      // Don't request permissions now, we'll ask later, after showing UI
      requestPermissions: false,
    })
    // It doesn't look like there is a registrationError being set for iOS.
    // https://github.com/zo0r/react-native-push-notification/issues/261
    PushNotificationIOS.addEventListener('registrationError', (error) => {
      emitter(({
        payload: {error},
        type: 'push:registrationError',
      }: PushConstants.PushRegistrationError))
    })

    console.log('Check push permissions')
    PushNotifications.checkPermissions(permissions => {
      console.log('Push checked permissions:', permissions)
      if (!permissions.alert) {
        // TODO(gabriel): Detect if we already showed permissions prompt and were denied,
        // in which case we should not show prompt or show different prompt about enabling
        // in Settings (for iOS)
        emitter(({
          payload: true,
          type: 'push:permissionsPrompt',
        }: PushConstants.PushPermissionsPromptAction))
      } else {
        // We have permissions, this triggers a token registration in
        // case it changed.
        emitter(({
          payload: undefined,
          type: 'push:permissionsRequest',
        }: PushConstants.PushPermissionsRequestAction))
      }
    })

    // TODO make some true unsubscribe function
    return () => {}
  })
}

export {
  requestPushPermissions,
  showMainWindow,
  configurePush,
}
