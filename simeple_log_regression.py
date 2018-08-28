import tensorflow as tf
import matplotlib.pyplot as plt
import numpy as np

import csv
import pandas as pd

# with open ('cpuutil.csv', 'r') as csvfile:
#     reader = csv.reader(csvfile)
#     for row in reader:
#         print(row)

data = pd.read_csv('cpuutil.csv', delimiter=",")

x = data['StartTime'].values       # shape (100, 1)
y = data['Average'].values          # shape (100, 1) + some noise

x = x.reshape(len(x),1)
y = y.reshape(len(yc

plt.scatter(x, y)
plt.show()

tf_x = tf.placeholder(tf.float32, x.shape)     # input x
tf_y = tf.placeholder(tf.float32, y.shape)

# print(x)
# print(x.shape)

# print(y)
# print(y.shape)

# neural network layers
l1 = tf.layers.dense(tf_x, 20, tf.nn.relu)          # hidden layer
output = tf.layers.dense(l1, 1)                     # output layer

loss = tf.losses.mean_squared_error(tf_y, output)   # compute cost
optimizer = tf.train.GradientDescentOptimizer(learning_rate=0.01)
train_op = optimizer.minimize(loss)

sess = tf.Session()                                 # control training and others
sess.run(tf.global_variables_initializer())         # initialize var in graph

plt.ion()   # something about plotting

for step in range(100):
    # train and net output
    _, l, pred = sess.run([train_op, loss, output], {tf_x: x, tf_y: y})
    #_, l, pred = sess.run([train_op, loss, output])

    print('step: %i/200' % step, '|train loss:', l, '|test loss:', pred)

    if step % 5 == 0:
        # plot and show learning process
        plt.cla()
        plt.scatter(x, y)
        plt.plot(x, pred, 'r-', lw=5)
        plt.text(0.5, 0, 'Loss=%.4f' % l, fontdict={'size': 20, 'color': 'red'})
        plt.pause(0.1)

plt.ioff()
plt.show()